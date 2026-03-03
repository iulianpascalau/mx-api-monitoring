package config

import (
	"testing"

	"github.com/pelletier/go-toml"
	"github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
	t.Parallel()

	testString := `
ListenAddress = "0.0.0.0:8080"
RetentionSeconds = 3600
StaticDir = "../../frontend/dist"
NumSecondsToConsiderStale = 300

[Alarms]
	Enabled = true
	NumSecondsLoopTimeAlarm = 60
	PushoverURL = "https://api.pushover.net/1/messages.json"
	TelegramURL = "https://api.telegram.org"
	NumRetries = 3
	SecondsBetweenRetries = 10
	[Alarms.SystemSelfCheck]
        Enabled = true
        DayOfWeek = "every day" # can also be "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday" and "Sunday"
        Hour = 12 # valid interval 0-23
        Minute = 0 # valid interval 0-59
        PollingIntervalInSec = 30
`

	expectedCfg := Config{
		ListenAddress:             "0.0.0.0:8080",
		RetentionSeconds:          3600,
		StaticDir:                 "../../frontend/dist",
		NumSecondsToConsiderStale: 300,
		Alarms: AlarmsConfig{
			Enabled:                 true,
			NumSecondsLoopTimeAlarm: 60,
			PushoverURL:             "https://api.pushover.net/1/messages.json",
			TelegramURL:             "https://api.telegram.org",
			NumRetries:              3,
			SecondsBetweenRetries:   10,
			SystemSelfCheck: SystemSelfCheckConfig{
				Enabled:              true,
				DayOfWeek:            "every day",
				Hour:                 12,
				Minute:               0,
				PollingIntervalInSec: 30,
			},
		},
	}

	cfg := Config{}

	err := toml.Unmarshal([]byte(testString), &cfg)
	assert.Nil(t, err)
	assert.Equal(t, expectedCfg, cfg)
}
