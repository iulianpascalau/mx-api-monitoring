package reporter

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/iulianpascalau/api-monitoring/services/agent/common"
	"github.com/iulianpascalau/api-monitoring/services/agent/config"
	"github.com/stretchr/testify/require"
)

func TestHTTPReporter_Report(t *testing.T) {
	var receivedBody string
	var receivedAuth string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "POST", r.Method)
		require.Equal(t, "application/json", r.Header.Get("Content-Type"))

		receivedAuth = r.Header.Get("X-Api-Key")

		buf := new(strings.Builder)
		_, _ = io.Copy(buf, r.Body)
		receivedBody = buf.String()

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	reporter := NewHTTPReporter(server.URL, "secret123", "AgentX", 2*time.Second)

	results := map[string]common.MetricResult{
		"Node1": {
			Config: config.EndpointConfig{Name: "Node1", Type: "uint64", NumAggregation: 10},
			Value:  "999",
		},
	}

	err := reporter.Report(context.Background(), results)
	require.NoError(t, err)

	require.Equal(t, "secret123", receivedAuth)
	require.Contains(t, receivedBody, `"AgentX.Active"`)
	require.Contains(t, receivedBody, `"Node1"`)
	require.Contains(t, receivedBody, `"999"`)
}
