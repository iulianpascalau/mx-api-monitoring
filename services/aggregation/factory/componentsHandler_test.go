package factory

import (
	"fmt"
	"testing"

	"github.com/iulianpascalau/api-monitoring/services/aggregation/common"
	"github.com/iulianpascalau/api-monitoring/services/aggregation/config"
	"github.com/multiversx/mx-chain-core-go/core/check"
	"github.com/stretchr/testify/assert"
)

func createMockEnvFileContents() map[string]string {
	return map[string]string{
		common.EnvServiceKey:       "service-key",
		common.EnvAuthUser:         "auth-user",
		common.EnvAuthPassword:     "auth-pass",
		common.EnvPushoverToken:    "pushover-token",
		common.EnvPushoverUserKey:  "pushover-userkey",
		common.EnvSMTPTo:           "smtp-to",
		common.EnvSMTPFrom:         "smtp-from",
		common.EnvSMTPPassword:     "smtp-pass",
		common.EnvSMTPPort:         "587",
		common.EnvSMTPHost:         "smtp-host",
		common.EnvTelegramBotToken: "telegram-bot",
		common.EnvTelegramChatId:   "telegram-chatid",
	}
}

func getMockConfig() config.Config {
	return config.Config{
		ListenAddress:             "0.0.0.0:0",
		RetentionSeconds:          3600,
		NumSecondsToConsiderStale: 300,
		Alarms: config.AlarmsConfig{
			Enabled:               true,
			PushoverURL:           "https://api.pushover.net/1/messages.json",
			TelegramURL:           "https://api.telegram.org",
			NumRetries:            3,
			SecondsBetweenRetries: 10,
			SystemSelfCheck: config.SystemSelfCheckConfig{
				Enabled:              true,
				DayOfWeek:            "every day",
				Hour:                 12,
				Minute:               0,
				PollingIntervalInSec: 30,
			},
		},
	}
}

func TestNewComponentsHandler(t *testing.T) {
	t.Parallel()

	handler, err := NewComponentsHandler(
		":memory:",
		createMockEnvFileContents(),
		getMockConfig(),
		log,
	)

	assert.NotNil(t, handler)
	assert.Nil(t, err)

	handler.Close()
}

func TestComponentsHandlerMethods(t *testing.T) {
	t.Parallel()

	t.Run("all components should be initialized", func(t *testing.T) {
		handler, _ := NewComponentsHandler(
			":memory:",
			createMockEnvFileContents(),
			getMockConfig(),
			log,
		)

		handler.Start()

		store := handler.GetStore()
		assert.Equal(t, "*storage.sqliteStorage", fmt.Sprintf("%T", store))

		serv := handler.GetServer()
		assert.Equal(t, "*api.server", fmt.Sprintf("%T", serv))

		assert.Equal(t, 4, len(handler.notifiers))
		assert.Equal(t, "*notifiers.logNotifier", fmt.Sprintf("%T", handler.notifiers[0]))
		assert.Equal(t, "*notifiers.pushoverNotifier", fmt.Sprintf("%T", handler.notifiers[1]))
		assert.Equal(t, "*notifiers.smtpNotifier", fmt.Sprintf("%T", handler.notifiers[2]))
		assert.Equal(t, "*notifiers.telegramNotifier", fmt.Sprintf("%T", handler.notifiers[3]))

		assert.False(t, check.IfNil(handler.pollingHandlerTrigger))

		handler.Close()
	})
	t.Run("no alarm components", func(t *testing.T) {
		cfg := getMockConfig()
		cfg.Alarms.Enabled = false

		handler, _ := NewComponentsHandler(
			":memory:",
			createMockEnvFileContents(),
			cfg,
			log,
		)

		handler.Start()

		store := handler.GetStore()
		assert.Equal(t, "*storage.sqliteStorage", fmt.Sprintf("%T", store))

		serv := handler.GetServer()
		assert.Equal(t, "*api.server", fmt.Sprintf("%T", serv))

		assert.Equal(t, 0, len(handler.notifiers))

		assert.True(t, check.IfNil(handler.pollingHandlerTrigger))

		handler.Close()
	})
	t.Run("alarm components without selfcheck", func(t *testing.T) {
		cfg := getMockConfig()
		cfg.Alarms.SystemSelfCheck.Enabled = false

		handler, _ := NewComponentsHandler(
			":memory:",
			createMockEnvFileContents(),
			cfg,
			log,
		)

		handler.Start()

		store := handler.GetStore()
		assert.Equal(t, "*storage.sqliteStorage", fmt.Sprintf("%T", store))

		serv := handler.GetServer()
		assert.Equal(t, "*api.server", fmt.Sprintf("%T", serv))

		assert.Equal(t, 4, len(handler.notifiers))
		assert.Equal(t, "*notifiers.logNotifier", fmt.Sprintf("%T", handler.notifiers[0]))
		assert.Equal(t, "*notifiers.pushoverNotifier", fmt.Sprintf("%T", handler.notifiers[1]))
		assert.Equal(t, "*notifiers.smtpNotifier", fmt.Sprintf("%T", handler.notifiers[2]))
		assert.Equal(t, "*notifiers.telegramNotifier", fmt.Sprintf("%T", handler.notifiers[3]))

		assert.True(t, check.IfNil(handler.pollingHandlerTrigger))

		handler.Close()
	})
}
