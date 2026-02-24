package poller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/iulianpascalau/api-monitoring/services/agent/config"
	"github.com/stretchr/testify/require"
)

func TestHTTPPoller_PollAll(t *testing.T) {
	// 1. Setup mock endpoint for successfully extracting JSON path
	successServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data": {"status": {"erd_nonce": 123456}}}`))
	}))
	defer successServer.Close()

	// 2. Setup mock endpoint that fails (Missing path)
	missingPathServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data": {"different_status": {"erd_nonce": 123456}}}`))
	}))
	defer missingPathServer.Close()

	// 3. Setup timeout server
	timeoutServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(3 * time.Second)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer timeoutServer.Close()

	endpoints := []config.EndpointConfig{
		{Name: "Node1", URL: successServer.URL, Value: "data.status.erd_nonce", Type: "uint64"},
		{Name: "Node2", URL: missingPathServer.URL, Value: "data.status.erd_nonce", Type: "uint64"},
		{Name: "Node3", URL: timeoutServer.URL, Value: "data.status.erd_nonce", Type: "uint64"},
		{Name: "Node4", URL: "http://localhost:59999", Value: "erd_nonce", Type: "uint64"}, // Connection Refused
	}

	// 1s timeout to trip Node3
	poller := NewHTTPPoller(1 * time.Second)
	ctx := context.Background()

	results := poller.PollAll(ctx, endpoints)

	// Since only Node1 succeeds, the map should be exactly size 1
	require.Len(t, results, 1)

	res, ok := results["Node1"]
	require.True(t, ok)
	require.Equal(t, "123456", res.Value)
	require.Equal(t, "uint64", res.Config.Type)
}
