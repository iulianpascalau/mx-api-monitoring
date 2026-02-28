package api

import (
	"context"

	"github.com/iulianpascalau/api-monitoring/services/aggregation/common"
)

// Storage defines the interface for persisting and querying metric data
type Storage interface {
	// SaveMetric updates the metric definition and appends a new value, trimming history to NumAggregation
	SaveMetric(ctx context.Context, name string, metricType string, numAggregation int, valString string, recordedAt int64) error

	// GetLatestMetrics returns the single latest recorded value for every known metric
	GetLatestMetrics(ctx context.Context) ([]common.MetricHistory, error)

	// GetMetricHistory returns the definition and all retained values (up to NumAggregation) for a specific metric
	GetMetricHistory(ctx context.Context, name string) (*common.MetricHistory, error)

	// DeleteMetric removes a metric definition and all associated values
	DeleteMetric(ctx context.Context, name string) error

	// UpdateMetricOrder updates the display order of a specific metric
	UpdateMetricOrder(ctx context.Context, name string, order int) error

	// UpdatePanelOrder updates the display order of a specific panel (VM)
	UpdatePanelOrder(ctx context.Context, name string, order int) error

	// GetPanelsConfigs returns the display configurations for all panels
	GetPanelsConfigs(ctx context.Context) (map[string]int, error)

	// Close shuts down the database connection
	Close() error

	IsInterfaceNil() bool
}
