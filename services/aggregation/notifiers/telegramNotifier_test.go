package notifiers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/iulianpascalau/api-monitoring/services/aggregation/common"
	logger "github.com/multiversx/mx-chain-logger-go"
	"github.com/stretchr/testify/assert"
)

const testTelegramToken = "test-token"
const testTelegramChatID = "test-chat-id"

func createHttpTestServerThatRespondsOKForTelegram(
	t *testing.T,
	expectedMessage string,
	expectedTitle string,
	numCalls *uint32,
) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		values := req.URL.Query()
		assert.Equal(t, testTelegramChatID, values.Get("chat_id"))
		assert.Equal(t, "html", values.Get("parse_mode"))
		assert.Contains(t, req.URL.Path, fmt.Sprintf("/bot%s/sendMessage", testTelegramToken))

		messageString := fmt.Sprintf("%s\n\n%s", expectedTitle, expectedMessage)
		assert.Equal(t, messageString, values.Get("text"))

		rw.WriteHeader(http.StatusOK)
		atomic.AddUint32(numCalls, 1)
	}))
}

func TestNewTelegramNotifier(t *testing.T) {
	t.Parallel()

	notifier := NewTelegramNotifier("url", "", "")
	assert.NotNil(t, notifier)
}

func TestTelegramNotifier_IsInterfaceNil(t *testing.T) {
	t.Parallel()

	var instance *telegramNotifier
	assert.True(t, instance.IsInterfaceNil())

	instance = &telegramNotifier{}
	assert.False(t, instance.IsInterfaceNil())
}

func TestTelegramNotifier_Name(t *testing.T) {
	t.Parallel()

	notifier := NewTelegramNotifier("url", "", "")
	assert.Equal(t, "*notifiers.telegramNotifier", notifier.Name())
}

func TestTelegramNotifier_OutputMessages(t *testing.T) {
	t.Parallel()

	t.Run("sending empty slice of messages should not call the service", func(t *testing.T) {
		t.Parallel()

		numCalls := uint32(0)
		expectedTitle := ""
		expectedMessage := ""
		testServer := createHttpTestServerThatRespondsOKForTelegram(t, expectedMessage, expectedTitle, &numCalls)
		defer testServer.Close()

		notifier := NewTelegramNotifier(testServer.URL, testTelegramToken, testTelegramChatID)
		err := notifier.OutputMessages()
		assert.Nil(t, err)

		time.Sleep(time.Second)
		assert.Equal(t, uint32(0), atomic.LoadUint32(&numCalls))
	})
	t.Run("post method fails should error", func(t *testing.T) {
		t.Parallel()

		notifier := NewTelegramNotifier("not-a-server-URL", "", "")
		err := notifier.OutputMessages(testInfoMessage)
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "not-a-server-URL")
	})
	t.Run("server errors should error", func(t *testing.T) {
		t.Parallel()

		testHttpServer := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			rw.WriteHeader(http.StatusInternalServerError)
		}))

		notifier := NewTelegramNotifier(testHttpServer.URL, "", "")
		err := notifier.OutputMessages(testInfoMessage)
		assert.ErrorIs(t, err, errReturnCodeIsNotOk)
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

		numCalls := uint32(0)
		expectedTitle := "ⓘ Info for executor"
		expectedMessage := `✅ info1: problem1

✅ info2

✅ info3

`

		testServer := createHttpTestServerThatRespondsOKForTelegram(t, expectedMessage, expectedTitle, &numCalls)
		defer testServer.Close()

		notifier := NewTelegramNotifier(testServer.URL, testTelegramToken, testTelegramChatID)
		err := notifier.OutputMessages(msg1, msg2, msg3)
		assert.Nil(t, err)

		time.Sleep(time.Second)
		assert.Equal(t, uint32(1), atomic.LoadUint32(&numCalls))
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

		numCalls := uint32(0)
		expectedTitle := "⚠️ Warnings occurred on executor"
		expectedMessage := `✅ info1: problem1

✅ info2

⚠️ warn1

`

		testServer := createHttpTestServerThatRespondsOKForTelegram(t, expectedMessage, expectedTitle, &numCalls)
		defer testServer.Close()

		notifier := NewTelegramNotifier(testServer.URL, testTelegramToken, testTelegramChatID)
		err := notifier.OutputMessages(msg1, msg2, msg3)
		assert.Nil(t, err)

		time.Sleep(time.Second)
		assert.Equal(t, uint32(1), atomic.LoadUint32(&numCalls))
	})
	t.Run("sending info, warn and error messages should work", func(t *testing.T) {
		t.Parallel()

		msg1 := common.OutputMessage{
			Type:               common.ErrorMessageOutputType,
			Identifier:         "err1",
			ExecutorName:       "executor",
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

		numCalls := uint32(0)
		expectedTitle := "🚨 Problems occurred on executor"
		expectedMessage := `🚨 err1: problem1

✅ info1

⚠️ warn1

`

		testServer := createHttpTestServerThatRespondsOKForTelegram(t, expectedMessage, expectedTitle, &numCalls)
		defer testServer.Close()

		notifier := NewTelegramNotifier(testServer.URL, testTelegramToken, testTelegramChatID)
		err := notifier.OutputMessages(msg1, msg2, msg3)
		assert.Nil(t, err)

		time.Sleep(time.Second)
		assert.Equal(t, uint32(1), atomic.LoadUint32(&numCalls))
	})
	t.Run("sending unknown type of messages should work", func(t *testing.T) {
		t.Parallel()

		msg1 := common.OutputMessage{
			Type:               0,
			Identifier:         "msg1",
			ExecutorName:       "executor",
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

		numCalls := uint32(0)
		expectedTitle := "executor"
		expectedMessage := ` msg1: problem1

 msg2

 msg3

`

		testServer := createHttpTestServerThatRespondsOKForTelegram(t, expectedMessage, expectedTitle, &numCalls)
		defer testServer.Close()

		notifier := NewTelegramNotifier(testServer.URL, testTelegramToken, testTelegramChatID)
		err := notifier.OutputMessages(msg1, msg2, msg3)
		assert.Nil(t, err)

		time.Sleep(time.Second)
		assert.Equal(t, uint32(1), atomic.LoadUint32(&numCalls))
	})
}

func TestTelegramNotifier_FunctionalTest(t *testing.T) {
	telegramBotToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	telegramChatID := os.Getenv("TELEGRAM_CHAT_ID")
	if len(telegramChatID) == 0 || len(telegramBotToken) == 0 {
		t.Skip("this is a functional test, will need real credentials. Please define your environment variables TELEGRAM_BOT_TOKEN and TELEGRAM_CHAT_ID so this test can work")
	}

	_ = logger.SetLogLevel("*:DEBUG")

	notifier := NewTelegramNotifier(
		"https://api.telegram.org",
		telegramBotToken,
		telegramChatID,
	)

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
