package config

import (
	"fmt"
	"os"

	"github.com/pelletier/go-toml/v2"
)

// Config maps to the config.toml file for the aggregation service
type Config struct {
	ListenAddress             string       `toml:"ListenAddress"`
	StaticDir                 string       `toml:"StaticDir"`
	RetentionSeconds          int          `toml:"RetentionSeconds"`
	NumSecondsToConsiderStale int          `toml:"NumSecondsToConsiderStale"`
	Alarms                    AlarmsConfig `toml:"Alarms"`
}

// AlarmsConfig defines the configuration for alarms
type AlarmsConfig struct {
	Enabled               bool                  `toml:"Enabled"`
	PushoverURL           string                `toml:"PushoverURL"`
	TelegramURL           string                `toml:"TelegramURL"`
	NumRetries            uint32                `toml:"NumRetries"`
	SecondsBetweenRetries int                   `toml:"SecondsBetweenRetries"`
	SystemSelfCheck       SystemSelfCheckConfig `toml:"SystemSelfCheck"`
}

// SystemSelfCheckConfig defines the configuration for the self check system
type SystemSelfCheckConfig struct {
	Enabled              bool
	DayOfWeek            string
	Hour                 int
	Minute               int
	PollingIntervalInSec int
}

// LoadConfig parses a TOML file into the Config struct
func LoadConfig(filepath string) (*Config, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file '%s': %w", filepath, err)
	}

	var cfg Config
	err = toml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to decode config file: %w", err)
	}

	return &cfg, nil
}
