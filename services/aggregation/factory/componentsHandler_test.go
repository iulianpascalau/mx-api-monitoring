package factory

import (
	"fmt"
	"testing"

	"github.com/iulianpascalau/api-monitoring/commonGo"
	"github.com/iulianpascalau/api-monitoring/services/aggregation/common"
	"github.com/iulianpascalau/api-monitoring/services/aggregation/config"
	"github.com/multiversx/mx-chain-core-go/core/check"
	"github.com/stretchr/testify/assert"
)

func createMockEnvFileContents() map[string]*commonGo.EnvValue {
	return map[string]*commonGo.EnvValue{
		common.EnvServiceKey:       {Value: "service-key"},
		common.EnvAuthUser:         {Value: "auth-user"},
		common.EnvAuthPassword:     {Value: "auth-pass"},
		common.EnvPushoverToken:    {Value: "pushover-token"},
		common.EnvPushoverUserKey:  {Value: "pushover-userkey"},
		common.EnvSMTPTo:           {Value: "smtp-to"},
		common.EnvSMTPFrom:         {Value: "smtp-from"},
		common.EnvSMTPPassword:     {Value: "smtp-pass"},
		common.EnvSMTPPort:         {Value: "587"},
		common.EnvSMTPHost:         {Value: "smtp-host"},
		common.EnvTelegramBotToken: {Value: "telegram-bot"},
		common.EnvTelegramChatId:   {Value: "telegram-chatid"},
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
			NumRetries:            1,
			SecondsBetweenRetries: 1,
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
		"test-version",
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
			"test-version",
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
			"test-version",
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
			"test-version",
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
