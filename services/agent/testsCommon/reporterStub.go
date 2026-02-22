package testsCommon

import (
	"context"

	"github.com/iulianpascalau/mx-api-monitoring/services/agent/common"
)

// ReporterStub -
type ReporterStub struct {
	ReportHandler func(ctx context.Context, results map[string]common.MetricResult) error
}

// Report -
func (stub *ReporterStub) Report(ctx context.Context, results map[string]common.MetricResult) error {
	if stub.ReportHandler != nil {
		return stub.ReportHandler(ctx, results)
	}

	return nil
}

// IsInterfaceNil -
func (stub *ReporterStub) IsInterfaceNil() bool {
	return stub == nil
}
