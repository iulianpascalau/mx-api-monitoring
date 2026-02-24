package api

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/iulianpascalau/api-monitoring/services/aggregation/common"
	"github.com/iulianpascalau/api-monitoring/services/aggregation/testsCommon"
	"github.com/stretchr/testify/require"
)

func TestNewServer_NilStorage(t *testing.T) {
	_, err := NewServer(ArgsWebServer{
		Storage:        nil,
		GeneralHandler: func(h http.Handler) http.Handler { return h },
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "storage is required")
}

func TestServer_StartAndClose(t *testing.T) {
	store := &testsCommon.StoreStub{}
	serv, err := NewServer(ArgsWebServer{
		ListenAddress:  "127.0.0.1:0", // random available port
		ServiceKeyApi:  "key",
		Storage:        store,
		GeneralHandler: func(h http.Handler) http.Handler { return h },
	})
	require.NoError(t, err)

	serv.Start()

	// Given it's a goroutine, allow a small time to boot
	time.Sleep(50 * time.Millisecond)

	err = serv.Close()
	require.NoError(t, err)
}

func TestHandlers_StorageErrors(t *testing.T) {
	store := &testsCommon.StoreStub{
		SaveMetricHandler: func(ctx context.Context, name string, metricType string, numAggregation int, valString string, recordedAt int64) error {
			return errors.New("db save error")
		},
		GetLatestMetricsHandler: func(ctx context.Context) ([]common.MetricHistory, error) {
			return nil, errors.New("db latest error")
		},
		GetMetricHistoryHandler: func(ctx context.Context, name string) (*common.MetricHistory, error) {
			return nil, errors.New("db hist error")
		},
		DeleteMetricHandler: func(ctx context.Context, name string) error {
			return errors.New("db del error")
		},
	}

	serv, err := NewServer(ArgsWebServer{
		ServiceKeyApi:  "test-secret",
		AuthUsername:   "admin",
		AuthPassword:   "password",
		ListenAddress:  ":0",
		Storage:        store,
		GeneralHandler: func(h http.Handler) http.Handler { return h },
	})
	require.NoError(t, err)

	token := getValidToken(serv)

	// handleReport (Storage Error is only logged, returns 200 OK since it processes in bulk)
	body := []byte(`{"metrics": {"VM1.CPU": {"value": "50", "type": "uint64", "numAggregation": 1}}}`)
	req, _ := http.NewRequest("POST", "/api/report", bytes.NewBuffer(body))
	req.Header.Set("X-Api-Key", "test-secret")
	w := httptest.NewRecorder()
	serv.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code) // The error is logged, handler doesn't fail the whole request

	// handleGetMetrics
	req, _ = http.NewRequest("GET", "/api/metrics", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	serv.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.Contains(t, w.Body.String(), "db latest error")

	// handleGetMetricHistory
	req, _ = http.NewRequest("GET", "/api/metrics/VM1.CPU/history", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	serv.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.Contains(t, w.Body.String(), "db hist error")

	// handleDeleteMetric
	req, _ = http.NewRequest("DELETE", "/api/metrics/VM1.CPU", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	serv.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.Contains(t, w.Body.String(), "db del error")
}

func TestHandlers_BadPayloads(t *testing.T) {
	store := &testsCommon.StoreStub{}
	serv, err := NewServer(ArgsWebServer{
		Storage:        store,
		GeneralHandler: func(h http.Handler) http.Handler { return h },
	})
	require.NoError(t, err)

	// Login with bad payload
	req, _ := http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer([]byte(`{bad-json}`)))
	w := httptest.NewRecorder()
	serv.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	// Login with wrong credentials
	req, _ = http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer([]byte(`{"username":"wrong", "password":"user"}`)))
	w = httptest.NewRecorder()
	serv.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthJWT_Errors(t *testing.T) {
	serv, err := NewServer(ArgsWebServer{
		Storage:        &testsCommon.StoreStub{},
		GeneralHandler: func(h http.Handler) http.Handler { return h },
	})
	require.NoError(t, err)

	tests := []struct {
		name   string
		header string
	}{
		{"Missing Token", ""},
		{"No Bearer Prefix", "invalid-token"},
		{"Invalid Token Parts", "Bearer header.payload"}, // missing signature
		{"Invalid Base64 Signature", "Bearer header.payload.$$$$$$$$"},
		{"Bad Signature Match", "Bearer ZXlKaGJHY2lPaUpJVXpJMU5pSXNJblI1Y0NJNklrcFhWQ0o5.eyJzdWIiOiJhZG1pbiIsImV4cCI6MTcxOTU4NTM1MH0.badsigbadsig"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/api/metrics", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			w := httptest.NewRecorder()
			serv.router.ServeHTTP(w, req)
			require.Equal(t, http.StatusUnauthorized, w.Code)
		})
	}
}
