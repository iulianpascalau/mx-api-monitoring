package reporter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/iulianpascalau/api-monitoring/services/agent/common"
	logger "github.com/multiversx/mx-chain-logger-go"
)

const activeHeartbeatName = "Active"
const separator = "."

var log = logger.GetOrCreate("reporter")

type httpReporter struct {
	endpoint string
	apiKey   string
	agentID  string
	client   *http.Client
}

// NewHTTPReporter creates a new reporter that pushes to the configured ReportEndpoint
func NewHTTPReporter(endpoint, apiKey, agentID string, timeout time.Duration) *httpReporter {
	return &httpReporter{
		endpoint: endpoint,
		apiKey:   apiKey,
		agentID:  agentID,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

func (r *httpReporter) Report(ctx context.Context, results map[string]common.MetricResult) error {
	payload := common.ReportPayload{
		Metrics: make(map[string]common.MetricPayload, len(results)+1), // +1 for heartbeat
	}

	for name, res := range results {
		payload.Metrics[name] = common.MetricPayload{
			Value:          res.Value,
			Type:           res.Config.Type,
			NumAggregation: res.Config.NumAggregation,
		}
	}

	// Always append heatbeat (agent metadata)
	payload.Metrics[r.agentID+separator+activeHeartbeatName] = common.MetricPayload{
		Value:          "true",
		Type:           "bool",
		NumAggregation: 1,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal report payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.endpoint, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create report request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Api-Key", r.apiKey)

	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("network error sending report: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("server rejected report with status code: %d", resp.StatusCode)
	}

	log.Debug("successfully sent metrics report", "endpoint", r.endpoint, "metrics_count", len(payload.Metrics))

	return nil
}

// IsInterfaceNil returns true if the value under the interface is nil
func (r *httpReporter) IsInterfaceNil() bool {
	return r == nil
}
