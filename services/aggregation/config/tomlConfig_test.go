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
RetentionSeconds =3600
`

	expectedCfg := Config{
		ListenAddress:    "0.0.0.0:8080",
		RetentionSeconds: 3600,
	}

	cfg := Config{}

	err := toml.Unmarshal([]byte(testString), &cfg)
	assert.Nil(t, err)
	assert.Equal(t, expectedCfg, cfg)
}
