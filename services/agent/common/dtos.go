package common

import "github.com/iulianpascalau/api-monitoring/services/agent/config"

// MetricResult holds the extracted value for a specific endpoint configuration
type MetricResult struct {
	Config config.EndpointConfig
	Value  string
}

// ReportPayload is the paylod to be sent to the reporting aggregation service
type ReportPayload struct {
	Metrics map[string]MetricPayload `json:"metrics"`
}

// MetricPayload defines a recorded metric value
type MetricPayload struct {
	Value          string `json:"value"`
	Type           string `json:"type"`
	NumAggregation int    `json:"numAggregation"`
}
