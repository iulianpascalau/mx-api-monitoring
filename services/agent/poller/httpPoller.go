package poller

import (
	"context"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/iulianpascalau/mx-api-monitoring/services/agent/common"
	"github.com/iulianpascalau/mx-api-monitoring/services/agent/config"
	logger "github.com/multiversx/mx-chain-logger-go"
	"github.com/tidwall/gjson"
)

var log = logger.GetOrCreate("poller")

type httpPoller struct {
	client *http.Client
}

// NewHTTPPoller creates a new HTTP-based poller with a default timeout
func NewHTTPPoller(timeout time.Duration) *httpPoller {
	return &httpPoller{
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// PollAll performs concurrent HTTP GETs to all configured endpoints and extracts exactly the JSON sub-path.
func (p *httpPoller) PollAll(ctx context.Context, endpoints []config.EndpointConfig) map[string]common.MetricResult {
	results := make(map[string]common.MetricResult)
	var mu sync.Mutex
	var wg sync.WaitGroup

	wg.Add(len(endpoints))
	for _, ep := range endpoints {
		go func(endpoint config.EndpointConfig) {
			defer wg.Done()

			val, err := p.pollEndpoint(ctx, endpoint)
			if err != nil {
				log.Warn("endpoint poll failed", "name", endpoint.Name, "url", endpoint.URL, "error", err)
				return // Omits from report
			}

			mu.Lock()
			results[endpoint.Name] = common.MetricResult{
				Config: endpoint,
				Value:  val,
			}
			mu.Unlock()
		}(ep)
	}

	wg.Wait()
	return results
}

func (p *httpPoller) pollEndpoint(ctx context.Context, ep config.EndpointConfig) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ep.URL, nil)
	if err != nil {
		return "", err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", errStatusNotOK(resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Use gjson to extract the path (e.g. "data.status.erd_nonce")
	result := gjson.GetBytes(body, ep.Value)
	if !result.Exists() {
		return "", errPathNotFound(ep.Value)
	}

	return result.String(), nil
}

// IsInterfaceNil returns true if the value under the interface is nil
func (p *httpPoller) IsInterfaceNil() bool {
	return p == nil
}
