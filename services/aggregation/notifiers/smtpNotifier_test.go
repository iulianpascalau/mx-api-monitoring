package notifiers

import (
	"errors"
	"fmt"
	"net/smtp"
	"os"
	"testing"

	"github.com/iulianpascalau/api-monitoring/services/aggregation/common"
	logger "github.com/multiversx/mx-chain-logger-go"
	"github.com/stretchr/testify/assert"
)

func TestNewSmtpNotifier(t *testing.T) {
	t.Parallel()

	notifier := NewSmtpNotifier(ArgsSmtpNotifier{})
	assert.NotNil(t, notifier)
}

func TestSmtpNotifier_IsInterfaceNil(t *testing.T) {
	t.Parallel()

	var instance *smtpNotifier
	assert.True(t, instance.IsInterfaceNil())

	instance = &smtpNotifier{}
	assert.False(t, instance.IsInterfaceNil())
}

func TestSmtpNotifier_Name(t *testing.T) {
	t.Parallel()

	notifier := NewSmtpNotifier(ArgsSmtpNotifier{})
	assert.Equal(t, "*notifiers.smtpNotifier", notifier.Name())
}

func TestSmtpNotifier_OutputMessages(t *testing.T) {
	testArgs := ArgsSmtpNotifier{
		To:       "to@email.com",
		SmtpPort: 37,
		SmtpHost: "host.email.com",
		From:     "from@email.com",
		Password: "pass",
	}
	expectedErr := errors.New("expected error")

	t.Run("sending empty slice of messages should not call the service", func(t *testing.T) {
		t.Parallel()

		notifier := NewSmtpNotifier(testArgs)
		notifier.sendMail = func(host string, auth smtp.Auth, from string, to []string, msgBytes []byte) error {
			assert.Fail(t, "should have not called sendMail function")

			return nil
		}
		err := notifier.OutputMessages()
		assert.Nil(t, err)
	})
	t.Run("send mail function fails, should error", func(t *testing.T) {
		t.Parallel()

		notifier := NewSmtpNotifier(testArgs)
		notifier.sendMail = func(host string, auth smtp.Auth, from string, to []string, msgBytes []byte) error {
			return expectedErr
		}
		err := notifier.OutputMessages(testInfoMessage)
		assert.NotNil(t, err)
		assert.ErrorIs(t, err, expectedErr)
	})
	t.Run("sending info messages should work", func(t *testing.T) {
		t.Parallel()

		msg1 := common.OutputMessage{
			Type:               common.InfoMessageOutputType,
			ExecutorName:       "executor",
			Identifier:         "info1",
			ProblemEncountered: "problem1",
		}
		msg2 := common.OutputMessage{
			Type:         common.InfoMessageOutputType,
			Identifier:   "info2",
			ExecutorName: "executor",
		}
		msg3 := common.OutputMessage{
			Type:         common.InfoMessageOutputType,
			Identifier:   "info3",
			ExecutorName: "executor",
		}

		expectedBody := `Subject: ⓘ Info for executor 
MIME-version: 1.0;
Content-Type: text/html; charset="UTF-8";




<!DOCTYPE html>
<html lang="en">
<body>
   ✅ info1: problem1

<br>✅ info2

<br>✅ info3

<br>
</body>
</html>
`
		var sentMsgBytes []byte
		notifier := NewSmtpNotifier(testArgs)
		notifier.sendMail = func(host string, auth smtp.Auth, from string, to []string, msgBytes []byte) error {
			assert.Equal(t, fmt.Sprintf("%s:%d", testArgs.SmtpHost, testArgs.SmtpPort), host)
			assert.Equal(t, testArgs.From, from)
			assert.Equal(t, []string{testArgs.To}, to)
			sentMsgBytes = msgBytes

			return nil
		}
		err := notifier.OutputMessages(msg1, msg2, msg3)
		assert.Nil(t, err)
		assert.Equal(t, expectedBody, string(sentMsgBytes))
	})
	t.Run("sending info messages and warn messages should work", func(t *testing.T) {
		t.Parallel()

		msg1 := common.OutputMessage{
			Type:               common.InfoMessageOutputType,
			ExecutorName:       "executor",
			Identifier:         "info1",
			ProblemEncountered: "problem1",
		}
		msg2 := common.OutputMessage{
			Type:         common.InfoMessageOutputType,
			Identifier:   "info2",
			ExecutorName: "executor",
		}
		msg3 := common.OutputMessage{
			Type:         common.WarningMessageOutputType,
			Identifier:   "warn1",
			ExecutorName: "executor",
		}

		expectedBody := `Subject: ⚠️ Warnings occurred on executor 
MIME-version: 1.0;
Content-Type: text/html; charset="UTF-8";




<!DOCTYPE html>
<html lang="en">
<body>
   ✅ info1: problem1

<br>✅ info2

<br>⚠️ warn1

<br>
</body>
</html>
`

		var sentMsgBytes []byte
		notifier := NewSmtpNotifier(testArgs)
		notifier.sendMail = func(host string, auth smtp.Auth, from string, to []string, msgBytes []byte) error {
			assert.Equal(t, fmt.Sprintf("%s:%d", testArgs.SmtpHost, testArgs.SmtpPort), host)
			assert.Equal(t, testArgs.From, from)
			assert.Equal(t, []string{testArgs.To}, to)
			sentMsgBytes = msgBytes

			return nil
		}
		err := notifier.OutputMessages(msg1, msg2, msg3)
		assert.Nil(t, err)
		assert.Equal(t, expectedBody, string(sentMsgBytes))
	})
	t.Run("sending info, warn and error messages should work", func(t *testing.T) {
		t.Parallel()

		msg1 := common.OutputMessage{
			Type:               common.ErrorMessageOutputType,
			ExecutorName:       "executor",
			Identifier:         "err1",
			ProblemEncountered: "problem1",
		}
		msg2 := common.OutputMessage{
			Type:         common.InfoMessageOutputType,
			Identifier:   "info1",
			ExecutorName: "executor",
		}
		msg3 := common.OutputMessage{
			Type:         common.WarningMessageOutputType,
			Identifier:   "warn1",
			ExecutorName: "executor",
		}

		expectedBody := `Subject: 🚨 Problems occurred on executor 
MIME-version: 1.0;
Content-Type: text/html; charset="UTF-8";




<!DOCTYPE html>
<html lang="en">
<body>
   🚨 err1: problem1

<br>✅ info1

<br>⚠️ warn1

<br>
</body>
</html>
`

		var sentMsgBytes []byte
		notifier := NewSmtpNotifier(testArgs)
		notifier.sendMail = func(host string, auth smtp.Auth, from string, to []string, msgBytes []byte) error {
			assert.Equal(t, fmt.Sprintf("%s:%d", testArgs.SmtpHost, testArgs.SmtpPort), host)
			assert.Equal(t, testArgs.From, from)
			assert.Equal(t, []string{testArgs.To}, to)
			sentMsgBytes = msgBytes

			return nil
		}
		err := notifier.OutputMessages(msg1, msg2, msg3)
		assert.Nil(t, err)
		assert.Equal(t, expectedBody, string(sentMsgBytes))
	})
	t.Run("sending unknown type of messages should work", func(t *testing.T) {
		t.Parallel()

		msg1 := common.OutputMessage{
			Type:               0,
			ExecutorName:       "executor",
			Identifier:         "msg1",
			ProblemEncountered: "problem1",
		}
		msg2 := common.OutputMessage{
			Type:         0,
			Identifier:   "msg2",
			ExecutorName: "executor",
		}
		msg3 := common.OutputMessage{
			Type:         0,
			Identifier:   "msg3",
			ExecutorName: "executor",
		}

		expectedBody := `Subject: executor 
MIME-version: 1.0;
Content-Type: text/html; charset="UTF-8";




<!DOCTYPE html>
<html lang="en">
<body>
    msg1: problem1

<br> msg2

<br> msg3

<br>
</body>
</html>
`

		var sentMsgBytes []byte
		notifier := NewSmtpNotifier(testArgs)
		notifier.sendMail = func(host string, auth smtp.Auth, from string, to []string, msgBytes []byte) error {
			assert.Equal(t, fmt.Sprintf("%s:%d", testArgs.SmtpHost, testArgs.SmtpPort), host)
			assert.Equal(t, testArgs.From, from)
			assert.Equal(t, []string{testArgs.To}, to)
			sentMsgBytes = msgBytes

			return nil
		}
		err := notifier.OutputMessages(msg1, msg2, msg3)
		assert.Nil(t, err)
		assert.Equal(t, expectedBody, string(sentMsgBytes))
	})
}

func TestSmtpNotifier_FunctionalTest(t *testing.T) {
	smtpTo := os.Getenv("SMTP_TO")
	smtpFrom := os.Getenv("SMTP_FROM")
	smtpPassword := os.Getenv("SMTP_PASSWORD")
	if len(smtpTo) == 0 || len(smtpFrom) == 0 || len(smtpPassword) == 0 {
		t.Skip("this is a functional test, will need real credentials. Please define your environment variables SMTP_TO, SMTP_FROM and SMTP_PASSWORD so this test can work")
	}

	_ = logger.SetLogLevel("*:DEBUG")

	args := ArgsSmtpNotifier{
		To:       smtpTo,
		SmtpPort: 587,
		SmtpHost: "smtp.gmail.com",
		From:     smtpFrom,
		Password: smtpPassword,
	}

	notifier := NewSmtpNotifier(args)

	t.Run("info messages", func(t *testing.T) {
		message1 := common.OutputMessage{
			Type:         common.InfoMessageOutputType,
			Identifier:   "this is an info line",
			ExecutorName: "API monitoring app",
		}
		message2 := common.OutputMessage{
			Type:         common.InfoMessageOutputType,
			Identifier:   "this is another info line",
			ExecutorName: "API monitoring app",
		}
		err := notifier.OutputMessages(message1, message2)
		assert.Nil(t, err)
	})
	t.Run("info and warn messages", func(t *testing.T) {
		message1 := common.OutputMessage{
			Type:         common.InfoMessageOutputType,
			Identifier:   "this is an info line",
			ExecutorName: "API monitoring app",
		}
		message2 := common.OutputMessage{
			Type:         common.InfoMessageOutputType,
			Identifier:   "this is another info line",
			ExecutorName: "API monitoring app",
		}
		message3 := common.OutputMessage{
			Type:         common.WarningMessageOutputType,
			Identifier:   "internal app errors occurred: 45",
			ExecutorName: "API monitoring app",
		}
		err := notifier.OutputMessages(message1, message2, message3)
		assert.Nil(t, err)
	})
	t.Run("error messages", func(t *testing.T) {
		message1 := common.OutputMessage{
			Type:               common.ErrorMessageOutputType,
			Identifier:         "VM host 0",
			ExecutorName:       "API monitoring app",
			ProblemEncountered: "Host appears offline",
		}
		message2 := common.OutputMessage{
			Type:               common.ErrorMessageOutputType,
			Identifier:         "VM host 1",
			ExecutorName:       "API monitoring app",
			ProblemEncountered: "Host appears offline",
		}
		err := notifier.OutputMessages(message1, message2)
		assert.Nil(t, err)
	})
}
