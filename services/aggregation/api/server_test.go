package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/iulianpascalau/mx-api-monitoring/services/aggregation/storage"
	"github.com/stretchr/testify/require"
)

func setupTestServer(t *testing.T) (*Server, Storage) {
	store, err := storage.NewSQLiteStorage(":memory:", 100)
	require.NoError(t, err)

	args := ArgsWebServer{
		ServiceKeyApi: "test-secret",
		AuthUsername:  "admin",
		AuthPassword:  "password",
		ListenAddress: ":0",
		Storage:       store,
	}

	server, err := NewServer(args)
	require.NoError(t, err)

	return server, store
}

func TestReportEndpoint(t *testing.T) {
	server, store := setupTestServer(t)
	defer func() {
		_ = store.Close()
	}()

	payload := MetricReportPayload{
		Metrics: map[string]struct {
			Value          string `json:"value"`
			Type           string `json:"type"`
			NumAggregation int    `json:"numAggregation"`
		}{
			"VM1.Active": {
				Value:          "true",
				Type:           "bool",
				NumAggregation: 1,
			},
		},
	}
	body, _ := json.Marshal(payload)

	// Test Unauthenticated
	req, _ := http.NewRequest("POST", "/api/report", bytes.NewBuffer(body))
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)

	// Test Authenticated
	req, _ = http.NewRequest("POST", "/api/report", bytes.NewBuffer(body))
	req.Header.Set("X-Api-Key", "test-secret")
	w = httptest.NewRecorder()
	server.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// Verify it reached DB
	metrics, err := store.GetLatestMetrics(context.Background())
	require.NoError(t, err)
	require.Len(t, metrics, 1)
	require.Equal(t, "VM1.Active", metrics[0].Name)
	require.Equal(t, "true", metrics[0].History[0].Value)
}

func TestLoginAndGetMetrics(t *testing.T) {
	server, store := setupTestServer(t)
	defer func() {
		_ = store.Close()
	}()

	// Seed DB
	err := store.SaveMetric(context.Background(), "VM1.Active", "bool", 1, "true", time.Now().Unix())
	require.NoError(t, err)

	// Test Login
	loginBody := `{"username":"admin", "password":"password"}`
	req, _ := http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer([]byte(loginBody)))
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var loginResp map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &loginResp)
	token := loginResp["token"]
	require.NotEmpty(t, token)

	// Test Get Metrics with Auth
	req, _ = http.NewRequest("GET", "/api/metrics", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	server.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var metricsResp struct {
		Metrics []struct {
			Name string `json:"name"`
		} `json:"metrics"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &metricsResp)
	require.Len(t, metricsResp.Metrics, 1)
	require.Equal(t, "VM1.Active", metricsResp.Metrics[0].Name)
}
