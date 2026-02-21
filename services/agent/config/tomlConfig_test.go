package config

import (
	"testing"

	"github.com/pelletier/go-toml"
	"github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
	t.Parallel()

	testString := `
Name = "VM1"
QueryIntervalInSeconds = 60
ReportEndpoint = "https://aaa.bbb.com/report"
ReportTimeoutInSeconds = 10

[[Endpoints]]
    Name = "VM1.Node1.nonce"
    URL = "http://127.0.0.1:8080/node/status"
    Value = "erd_nonce"
    Type = "uint64"
    NumAggregation = 100

[[Endpoints]]
    Name = "VM1.Node2.nonce"
    URL = "http://127.0.0.1:8081/node/status"
    Value = "erd_nonce"
    Type = "uint64"
    NumAggregation = 100

[[Endpoints]]
    Name = "VM1.Node1.epoch"
    URL = "http://127.0.0.1:8080/network/status"
    Value = "erd_epoch_number"
    Type = "uint64"
    NumAggregation = 1
`

	expectedCfg := Config{
		Name:                   "VM1",
		QueryIntervalInSeconds: 60,
		ReportEndpoint:         "https://aaa.bbb.com/report",
		ReportTimeoutInSeconds: 10,
		Endpoints: []EndpointConfig{
			{
				Name:           "VM1.Node1.nonce",
				URL:            "http://127.0.0.1:8080/node/status",
				Value:          "erd_nonce",
				Type:           "uint64",
				NumAggregation: 100,
			},
			{
				Name:           "VM1.Node2.nonce",
				URL:            "http://127.0.0.1:8081/node/status",
				Value:          "erd_nonce",
				Type:           "uint64",
				NumAggregation: 100,
			},
			{
				Name:           "VM1.Node1.epoch",
				URL:            "http://127.0.0.1:8080/network/status",
				Value:          "erd_epoch_number",
				Type:           "uint64",
				NumAggregation: 1,
			},
		},
	}

	cfg := Config{}

	err := toml.Unmarshal([]byte(testString), &cfg)
	assert.Nil(t, err)
	assert.Equal(t, expectedCfg, cfg)
}
