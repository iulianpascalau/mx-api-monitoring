package notifiers

import (
	"testing"

	"github.com/iulianpascalau/api-monitoring/services/aggregation/common"
	"github.com/iulianpascalau/api-monitoring/services/aggregation/testsCommon"
	logger "github.com/multiversx/mx-chain-logger-go"
	"github.com/stretchr/testify/assert"
)

func TestNewLogNotifier(t *testing.T) {
	t.Parallel()

	t.Run("nil logger should error", func(t *testing.T) {
		t.Parallel()

		notifier, err := NewLogNotifier(nil)
		assert.Nil(t, notifier)
		assert.Equal(t, errNilLogger, err)
	})
	t.Run("should work", func(t *testing.T) {
		t.Parallel()

		notifier, err := NewLogNotifier(&testsCommon.LoggerStub{})
		assert.NotNil(t, notifier)
		assert.Nil(t, err)
	})
}

func TestLogNotifier_IsInterfaceNil(t *testing.T) {
	t.Parallel()

	var notifier *logNotifier
	assert.True(t, notifier.IsInterfaceNil())

	notifier = &logNotifier{}
	assert.False(t, notifier.IsInterfaceNil())
}

func TestLogNotifier_Name(t *testing.T) {
	t.Parallel()

	notifier, _ := NewLogNotifier(&testsCommon.LoggerStub{})
	assert.Equal(t, "*notifiers.logNotifier", notifier.Name())
}

func TestLogNotifier_OutputErrorMessages(t *testing.T) {
	t.Parallel()

	messages := make(map[string]logger.LogLevel)
	logInstance := &testsCommon.LoggerStub{
		LogHandler: func(logLevel logger.LogLevel, message string, args ...interface{}) {
			messages[message] = logLevel
		},
	}

	notifier, _ := NewLogNotifier(logInstance)
	message1 := common.OutputMessage{
		Type:               common.ErrorMessageOutputType,
		Identifier:         "vm host 0",
		ExecutorName:       "executor",
		ProblemEncountered: "Host appears offline",
	}

	message2 := common.OutputMessage{
		Type:               common.WarningMessageOutputType,
		Identifier:         "vm host 1",
		ExecutorName:       "executor",
		ProblemEncountered: "Host appears offline",
	}

	message3 := common.OutputMessage{
		Type:               common.InfoMessageOutputType,
		Identifier:         "vm host 2",
		ExecutorName:       "executor",
		ProblemEncountered: "Host appears offline",
	}

	err := notifier.OutputMessages(message1, message2, message3)
	assert.Nil(t, err)

	expectedMap := map[string]logger.LogLevel{
		"vm host 0 -> Host appears offline called by executor": logger.LogError,
		"vm host 1 -> Host appears offline called by executor": logger.LogWarning,
		"vm host 2 -> Host appears offline called by executor": logger.LogInfo,
	}

	assert.Equal(t, expectedMap, messages)
}
