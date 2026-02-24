package e2e_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	agentCfg "github.com/iulianpascalau/mx-api-monitoring/services/agent/config"
	agentFactory "github.com/iulianpascalau/mx-api-monitoring/services/agent/factory"
	aggCfg "github.com/iulianpascalau/mx-api-monitoring/services/aggregation/config"
	aggFactory "github.com/iulianpascalau/mx-api-monitoring/services/aggregation/factory"
	logger "github.com/multiversx/mx-chain-logger-go"
	"github.com/stretchr/testify/require"
)

var log = logger.GetOrCreate("e2e-test")

func TestE2EFlow(t *testing.T) {
	log.Info("======== 1. Start a mock target API that the Agent will monitor")
	mockAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// We'll mimic a JSON payload where `status` is what we want
		_, _ = w.Write([]byte(`{"status": "ok", "latency": 12}`))
	}))
	defer mockAPI.Close()

	log.Info("======== 2. Prepare SQLite path for Aggregation")
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "e2e_sqlite.db")

	log.Info("======== 3. Start Aggregation Service via componentsHandler")
	aggregationConfig := aggCfg.Config{
		ListenAddress:    "127.0.0.1:0",
		RetentionSeconds: 3600,
	}

	aggregationHandler, err := aggFactory.NewComponentsHandler(
		dbPath,
		"test-service-key",
		"admin",
		"password",
		aggregationConfig,
	)
	require.NoError(t, err)

	aggregationHandler.Start()
	defer aggregationHandler.Close()

	_, port, err := net.SplitHostPort(aggregationHandler.GetServer().Address())
	require.NoError(t, err)
	aggURL := fmt.Sprintf("http://127.0.0.1:%s", port)

	log.Info("======== 3.1. Wait a moment for server to start")
	time.Sleep(100 * time.Millisecond)

	log.Info("======== 4. Start Agent Service via componentsHandler")
	agentConfig := agentCfg.Config{
		Name:                   "e2e-agent",
		QueryIntervalInSeconds: 1,
		ReportEndpoint:         aggURL + "/api/report",
		ReportTimeoutInSeconds: 5,
		Endpoints: []agentCfg.EndpointConfig{
			{
				Name:           "mock-api",
				URL:            mockAPI.URL,
				Value:          "status",
				Type:           "string",
				NumAggregation: 1,
			},
		},
	}

	agentHandler, err := agentFactory.NewComponentsHandler(
		"test-service-key",
		agentConfig,
	)
	require.NoError(t, err)

	agentHandler.Start()
	defer agentHandler.Close()

	log.Info("======== 5. Wait for agent to poll the mockAPI and report to Aggregation")
	// Agent queries every 1s, we'll wait about 2.5s to ensure at least 2 queries
	time.Sleep(2500 * time.Millisecond)

	log.Info("======== 6. Test the Aggregation API using HTTP calls")
	log.Info("======== 6.a. Login to get JWT")
	loginBody, _ := json.Marshal(map[string]string{
		"username": "admin",
		"password": "password",
	})
	respLogin, err := http.Post(aggURL+"/api/auth/login", "application/json", bytes.NewBuffer(loginBody))
	require.NoError(t, err)
	defer func() {
		_ = respLogin.Body.Close()
	}()
	require.Equal(t, http.StatusOK, respLogin.StatusCode)

	var loginData struct {
		Token string `json:"token"`
	}
	err = json.NewDecoder(respLogin.Body).Decode(&loginData)
	require.NoError(t, err)
	require.NotEmpty(t, loginData.Token)

	log.Info("======== 6.b. Fetch Metrics")
	reqMetrics, err := http.NewRequest(http.MethodGet, aggURL+"/api/metrics", nil)
	require.NoError(t, err)
	reqMetrics.Header.Set("Authorization", "Bearer "+loginData.Token)

	client := &http.Client{}
	respMetrics, err := client.Do(reqMetrics)
	require.NoError(t, err)
	defer func() {
		_ = respMetrics.Body.Close()
	}()
	require.Equal(t, http.StatusOK, respMetrics.StatusCode)

	var metricsData struct {
		Metrics []struct {
			Name           string `json:"name"`
			Value          string `json:"value"`
			Type           string `json:"type"`
			NumAggregation int    `json:"numAggregation"`
		} `json:"metrics"`
	}
	b, _ := io.ReadAll(respMetrics.Body)
	err = json.Unmarshal(b, &metricsData)
	require.NoError(t, err)

	log.Info("======== 6.c. Verify our metric is present")
	require.NotEmpty(t, metricsData.Metrics, "Expected metrics to be present")
	found := false
	for _, m := range metricsData.Metrics {
		if m.Name == "mock-api" {
			found = true
			require.Equal(t, "ok", m.Value)
			require.Equal(t, "string", m.Type)
			require.Equal(t, 1, m.NumAggregation)
		}
	}
	require.True(t, found, "Expected to find mock-api metric")

	log.Info("======== 6.d. Fetch Metric History")
	reqHistory, err := http.NewRequest(http.MethodGet, aggURL+"/api/metrics/mock-api/history", nil)
	require.NoError(t, err)
	reqHistory.Header.Set("Authorization", "Bearer "+loginData.Token)

	respHistory, err := client.Do(reqHistory)
	require.NoError(t, err)
	defer func() {
		_ = respHistory.Body.Close()
	}()
	require.Equal(t, http.StatusOK, respHistory.StatusCode)

	var historyData struct {
		Name           string `json:"name"`
		Type           string `json:"type"`
		NumAggregation int    `json:"numAggregation"`
		History        []struct {
			Value      string `json:"value"`
			RecordedAt int64  `json:"recordedAt"`
		} `json:"history"`
	}
	h, _ := io.ReadAll(respHistory.Body)
	err = json.Unmarshal(h, &historyData)
	require.NoError(t, err)
	require.Equal(t, "mock-api", historyData.Name)
	require.NotEmpty(t, historyData.History)
	require.Equal(t, "ok", historyData.History[0].Value)

	log.Info("======== 6.e. Delete Metric")
	reqDelete, err := http.NewRequest(http.MethodDelete, aggURL+"/api/metrics/mock-api", nil)
	require.NoError(t, err)
	reqDelete.Header.Set("Authorization", "Bearer "+loginData.Token)

	respDelete, err := client.Do(reqDelete)
	require.NoError(t, err)
	defer func() {
		_ = respDelete.Body.Close()
	}()
	require.Equal(t, http.StatusOK, respDelete.StatusCode)

	log.Info("======== 6.f. Verify deletion")
	reqHistoryAfter, err := http.NewRequest(http.MethodGet, aggURL+"/api/metrics/mock-api/history", nil)
	require.NoError(t, err)
	reqHistoryAfter.Header.Set("Authorization", "Bearer "+loginData.Token)

	respHistoryAfter, err := client.Do(reqHistoryAfter)
	require.NoError(t, err)
	defer func() {
		_ = respHistoryAfter.Body.Close()
	}()
	require.Equal(t, http.StatusNotFound, respHistoryAfter.StatusCode)
}
