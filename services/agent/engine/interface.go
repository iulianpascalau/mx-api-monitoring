package engine

import (
	"context"

	"github.com/iulianpascalau/api-monitoring/services/agent/common"
	"github.com/iulianpascalau/api-monitoring/services/agent/config"
)

// Poller defines the interface for fetching values from local endpoints
type Poller interface {
	// PollAll performs concurrent HTTP GETs to all configured endpoints and extracts exactly the JSON sub-path.
	// Endpoints that fail/timeout or lack the JSON path are omitted from the returned map.
	PollAll(ctx context.Context, endpoints []config.EndpointConfig) map[string]common.MetricResult

	IsInterfaceNil() bool
}

// Reporter defines the interface for pushing polled metrics to the aggregation service
type Reporter interface {
	// Report sends a payload containing the polled results and a heartbeat to the server.
	// Specs state reporting failures log an error and omit immediate retry.
	Report(ctx context.Context, results map[string]common.MetricResult) error

	IsInterfaceNil() bool
}
