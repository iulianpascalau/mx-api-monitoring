package config

import (
	"fmt"
	"os"

	"github.com/pelletier/go-toml/v2"
)

// EndpointConfig defines a single metric polling rule
type EndpointConfig struct {
	Name           string `toml:"Name"`
	URL            string `toml:"URL"`
	Value          string `toml:"Value"`
	Type           string `toml:"Type"`
	NumAggregation int    `toml:"NumAggregation"`
}

// Config maps to the config.toml file for the monitor agent
type Config struct {
	Name                   string           `toml:"Name"`
	QueryIntervalInSeconds uint32           `toml:"QueryIntervalInSeconds"`
	ReportEndpoint         string           `toml:"ReportEndpoint"`
	ReportTimeoutInSeconds uint32           `toml:"ReportTimeoutInSeconds"`
	Endpoints              []EndpointConfig `toml:"Endpoints"`
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
