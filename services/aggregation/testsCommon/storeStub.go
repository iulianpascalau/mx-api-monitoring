package testsCommon

import (
	"context"

	"github.com/iulianpascalau/api-monitoring/services/aggregation/common"
)

// StoreStub -
type StoreStub struct {
	SaveMetricHandler        func(ctx context.Context, name string, metricType string, numAggregation int, valString string, recordedAt int64) error
	GetLatestMetricsHandler  func(ctx context.Context) ([]common.MetricHistory, error)
	GetMetricHistoryHandler  func(ctx context.Context, name string) (*common.MetricHistory, error)
	DeleteMetricHandler      func(ctx context.Context, name string) error
	UpdateMetricOrderHandler func(ctx context.Context, name string, order int) error
	UpdatePanelOrderHandler  func(ctx context.Context, name string, order int) error
	GetPanelsConfigsHandler  func(ctx context.Context) (map[string]int, error)
	CloseHandler             func() error
}

// SaveMetric -
func (stub *StoreStub) SaveMetric(ctx context.Context, name string, metricType string, numAggregation int, valString string, recordedAt int64) error {
	if stub.SaveMetricHandler != nil {
		return stub.SaveMetricHandler(ctx, name, metricType, numAggregation, valString, recordedAt)
	}

	return nil
}

// GetLatestMetrics -
func (stub *StoreStub) GetLatestMetrics(ctx context.Context) ([]common.MetricHistory, error) {
	if stub.GetLatestMetricsHandler != nil {
		return stub.GetLatestMetricsHandler(ctx)
	}

	return make([]common.MetricHistory, 0), nil
}

// GetMetricHistory -
func (stub *StoreStub) GetMetricHistory(ctx context.Context, name string) (*common.MetricHistory, error) {
	if stub.GetMetricHistoryHandler != nil {
		return stub.GetMetricHistoryHandler(ctx, name)
	}

	return &common.MetricHistory{}, nil
}

// DeleteMetric -
func (stub *StoreStub) DeleteMetric(ctx context.Context, name string) error {
	if stub.DeleteMetricHandler != nil {
		return stub.DeleteMetricHandler(ctx, name)
	}

	return nil
}

// UpdateMetricOrder -
func (stub *StoreStub) UpdateMetricOrder(ctx context.Context, name string, order int) error {
	if stub.UpdateMetricOrderHandler != nil {
		return stub.UpdateMetricOrderHandler(ctx, name, order)
	}

	return nil
}

// UpdatePanelOrder -
func (stub *StoreStub) UpdatePanelOrder(ctx context.Context, name string, order int) error {
	if stub.UpdatePanelOrderHandler != nil {
		return stub.UpdatePanelOrderHandler(ctx, name, order)
	}

	return nil
}

// GetPanelsConfigs -
func (stub *StoreStub) GetPanelsConfigs(ctx context.Context) (map[string]int, error) {
	if stub.GetPanelsConfigsHandler != nil {
		return stub.GetPanelsConfigsHandler(ctx)
	}

	return make(map[string]int), nil
}

// Close -
func (stub *StoreStub) Close() error {
	if stub.CloseHandler != nil {
		return stub.CloseHandler()
	}

	return nil
}

// IsInterfaceNil -
func (stub *StoreStub) IsInterfaceNil() bool {
	return stub == nil
}
