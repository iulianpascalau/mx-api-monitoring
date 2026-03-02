package notifiers

import (
	"encoding/json"
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

const testToken = "test-token"
const testUserKey = "test-user-key"

var testInfoMessage = common.OutputMessage{
	Type:               common.InfoMessageOutputType,
	ExecutorName:       "executor",
	Identifier:         "info2",
	ProblemEncountered: "problem1",
}

func createHttpTestServerThatRespondsOK(
	t *testing.T,
	expectedMessage string,
	expectedTitle string,
	numCalls *uint32,
) *httptest.Server {
	response := &pushoverResponse{
		Status:  1,
		Request: "e43a9e0f-6836-42f1-8b06-e8bc56012637",
	}
	responseBytes, _ := json.Marshal(response)

	return httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		body := req.Body
		defer func() {
			errClose := body.Close()
			assert.Nil(t, errClose)
		}()
		buff := make([]byte, 524288)
		numRead, _ := body.Read(buff)
		buff = buff[:numRead]

		request := &pushoverRequest{}
		err := json.Unmarshal(buff, request)
		assert.Nil(t, err)

		assert.Equal(t, testToken, request.Token)
		assert.Equal(t, testUserKey, request.User)
		assert.Equal(t, 1, request.HTML)
		assert.Equal(t, expectedMessage, request.Message)
		assert.Equal(t, expectedTitle, request.Title)

		rw.WriteHeader(http.StatusOK)
		_, _ = rw.Write(responseBytes)
		atomic.AddUint32(numCalls, 1)
	}))
}

func TestNewPushoverNotifier(t *testing.T) {
	t.Parallel()

	notifier := NewPushoverNotifier("url", "", "")
	assert.NotNil(t, notifier)
}

func TestPushoverNotifier_IsInterfaceNil(t *testing.T) {
	t.Parallel()

	var instance *pushoverNotifier
	assert.True(t, instance.IsInterfaceNil())

	instance = &pushoverNotifier{}
	assert.False(t, instance.IsInterfaceNil())
}

func TestPushoverNotifier_Name(t *testing.T) {
	t.Parallel()

	notifier := NewPushoverNotifier("url", "", "")
	assert.Equal(t, "*notifiers.pushoverNotifier", notifier.Name())
}

func TestPushoverNotifier_OutputMessages(t *testing.T) {
	t.Parallel()

	t.Run("sending empty slice of messages should not call the service", func(t *testing.T) {
		t.Parallel()

		numCalls := uint32(0)
		expectedTitle := ""
		expectedMessage := ""
		testServer := createHttpTestServerThatRespondsOK(t, expectedMessage, expectedTitle, &numCalls)
		defer testServer.Close()

		notifier := NewPushoverNotifier(testServer.URL, testToken, testUserKey)
		err := notifier.OutputMessages()
		assert.Nil(t, err)

		time.Sleep(time.Second)
		assert.Equal(t, uint32(0), atomic.LoadUint32(&numCalls))
	})
	t.Run("post method fails should error", func(t *testing.T) {
		t.Parallel()

		notifier := NewPushoverNotifier("not-a-server-URL", "", "")
		err := notifier.OutputMessages(testInfoMessage)
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "not-a-server-URL")
	})
	t.Run("server errors should error", func(t *testing.T) {
		t.Parallel()

		testHttpServer := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			rw.WriteHeader(http.StatusInternalServerError)
		}))

		notifier := NewPushoverNotifier(testHttpServer.URL, "", "")
		err := notifier.OutputMessages(testInfoMessage)
		assert.ErrorIs(t, err, errReturnCodeIsNotOk)
	})
	t.Run("http post response is not OK, should error", func(t *testing.T) {
		t.Parallel()

		testHttpServer := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			rw.WriteHeader(http.StatusOK)
			_, _ = rw.Write([]byte("not-a-valid-json"))
		}))

		notifier := NewPushoverNotifier(testHttpServer.URL, "", "")
		err := notifier.OutputMessages(testInfoMessage)
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "invalid character")

		// make sure any accidental calls on API endpoint routes are caught by the test server
		time.Sleep(time.Second)
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

		testServer := createHttpTestServerThatRespondsOK(t, expectedMessage, expectedTitle, &numCalls)
		defer testServer.Close()

		notifier := NewPushoverNotifier(testServer.URL, testToken, testUserKey)
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

		testServer := createHttpTestServerThatRespondsOK(t, expectedMessage, expectedTitle, &numCalls)
		defer testServer.Close()

		notifier := NewPushoverNotifier(testServer.URL, testToken, testUserKey)
		err := notifier.OutputMessages(msg1, msg2, msg3)
		assert.Nil(t, err)

		time.Sleep(time.Second)
		assert.Equal(t, uint32(1), atomic.LoadUint32(&numCalls))
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

		numCalls := uint32(0)
		expectedTitle := "🚨 Problems occurred on executor"
		expectedMessage := `🚨 err1: problem1

✅ info1

⚠️ warn1

`

		testServer := createHttpTestServerThatRespondsOK(t, expectedMessage, expectedTitle, &numCalls)
		defer testServer.Close()

		notifier := NewPushoverNotifier(testServer.URL, testToken, testUserKey)
		err := notifier.OutputMessages(msg1, msg2, msg3)
		assert.Nil(t, err)

		time.Sleep(time.Second)
		assert.Equal(t, uint32(1), atomic.LoadUint32(&numCalls))
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

		numCalls := uint32(0)
		expectedTitle := "executor"
		expectedMessage := ` msg1: problem1

 msg2

 msg3

`

		testServer := createHttpTestServerThatRespondsOK(t, expectedMessage, expectedTitle, &numCalls)
		defer testServer.Close()

		notifier := NewPushoverNotifier(testServer.URL, testToken, testUserKey)
		err := notifier.OutputMessages(msg1, msg2, msg3)
		assert.Nil(t, err)

		time.Sleep(time.Second)
		assert.Equal(t, uint32(1), atomic.LoadUint32(&numCalls))
	})
}

func TestPushoverNotifier_FunctionalTest(t *testing.T) {
	pushoverToken := os.Getenv("PUSHOVER_TOKEN")
	pushoverUserKey := os.Getenv("PUSHOVER_USERKEY")
	if len(pushoverToken) == 0 || len(pushoverUserKey) == 0 {
		t.Skip("this is a functional test, will need real credentials. Please define your environment variables PUSHOVER_TOKEN and PUSHOVER_USERKEY so this test can work")
	}

	_ = logger.SetLogLevel("*:DEBUG")

	notifier := NewPushoverNotifier(
		"https://api.pushover.net/1/messages.json",
		pushoverToken,
		pushoverUserKey,
	)

	t.Run("info messages", func(t *testing.T) {
		message1 := common.OutputMessage{
			Type:         common.InfoMessageOutputType,
			Identifier:   "this is an info line",
			ExecutorName: "API monitoring app",
		}
		message2 := common.OutputMessage{
			Type:         common.InfoMessageOutputType,
			Identifier:   "this is an info line",
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
			Identifier:   "this is an info line",
			ExecutorName: "API monitoring app",
		}
		message3 := common.OutputMessage{
			Type:         common.WarningMessageOutputType,
			Identifier:   "this is an info line",
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
