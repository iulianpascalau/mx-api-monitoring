package testCommon

import (
	"context"

	"github.com/iulianpascalau/mx-api-monitoring/services/agent/common"
	"github.com/iulianpascalau/mx-api-monitoring/services/agent/config"
)

// PollerStub -
type PollerStub struct {
	PollAllHandler func(ctx context.Context, endpoints []config.EndpointConfig) map[string]common.MetricResult
}

// PollAll -
func (stub *PollerStub) PollAll(ctx context.Context, endpoints []config.EndpointConfig) map[string]common.MetricResult {
	if stub.PollAllHandler != nil {
		return stub.PollAllHandler(ctx, endpoints)
	}

	return make(map[string]common.MetricResult)
}

// IsInterfaceNil -
func (stub *PollerStub) IsInterfaceNil() bool {
	return stub == nil
}
