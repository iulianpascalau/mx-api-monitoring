package common

// MetricDefinition defines the structure of a metric in the metrics table
type MetricDefinition struct {
	Name           string `json:"name"`
	Type           string `json:"type"`
	NumAggregation int    `json:"numAggregation"`
}

// MetricValue represents a single reported data point
type MetricValue struct {
	Value      string `json:"value"` // Stored natively in DB but returned as string to API
	RecordedAt int64  `json:"recordedAt"`
}

// MetricHistory encapsulates a metric's definition and its recent time-series values
type MetricHistory struct {
	Name           string        `json:"name"`
	Type           string        `json:"type"`
	NumAggregation int           `json:"numAggregation"`
	History        []MetricValue `json:"history"`
}
