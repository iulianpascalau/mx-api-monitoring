package storage

import (
	"context"
	"testing"
	"time"

	"github.com/iulianpascalau/api-monitoring/services/aggregation/common"
	"github.com/stretchr/testify/require"
)

func TestSQLiteStorage_SaveAndGet(t *testing.T) {
	s, err := NewSQLiteStorage(":memory:", 3600)
	require.NoError(t, err)
	require.False(t, s.IsInterfaceNil())
	defer func() {
		_ = s.Close()
	}()

	ctx := context.Background()
	now := time.Now().Unix()

	// 1. Save uint64 metric
	err = s.SaveMetric(ctx, "VM1.Node1.nonce", "uint64", 2, "100", now-10)
	require.NoError(t, err)

	err = s.SaveMetric(ctx, "VM1.Node1.nonce", "uint64", 2, "101", now-5)
	require.NoError(t, err)

	// Will cause trimming of "100" (aggregation is 2)
	err = s.SaveMetric(ctx, "VM1.Node1.nonce", "uint64", 2, "102", now)
	require.NoError(t, err)

	// 2. Save bool metric
	err = s.SaveMetric(ctx, "VM1.Active", "bool", 1, "true", now)
	require.NoError(t, err)

	// Retrieve History
	hist, err := s.GetMetricHistory(ctx, "VM1.Node1.nonce")
	require.NoError(t, err)
	require.Equal(t, 2, len(hist.History))
	require.Equal(t, "101", hist.History[0].Value) // ascending timestamp order
	require.Equal(t, "102", hist.History[1].Value)

	// Retrieve Latest
	latest, err := s.GetLatestMetrics(ctx)
	require.NoError(t, err)
	// Should return 2 metrics, order is not guaranteed due to lack of order by in GetLatestMetrics query wrapper
	var activeVal *common.MetricHistory
	var nonceVal *common.MetricHistory
	for i := range latest {
		if latest[i].Name == "VM1.Active" {
			activeVal = &latest[i]
		}
		if latest[i].Name == "VM1.Node1.nonce" {
			nonceVal = &latest[i]
		}
	}

	require.NotNil(t, activeVal)
	require.Equal(t, "true", activeVal.History[0].Value)

	require.NotNil(t, nonceVal)
	require.Equal(t, "102", nonceVal.History[0].Value)

	// Test deletion
	err = s.DeleteMetric(ctx, "VM1.Active")
	require.NoError(t, err)

	latestAfterDelete, err := s.GetLatestMetrics(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, len(latestAfterDelete))
	require.Equal(t, "VM1.Node1.nonce", latestAfterDelete[0].Name)
}

func TestSQLiteStorage_RetentionCleaner(t *testing.T) {
	// Set retention very low (3 seconds) to trigger cleaner fast in memory
	s, err := NewSQLiteStorage(":memory:", 3)
	require.NoError(t, err)
	defer func() {
		_ = s.Close()
	}()

	ctx := context.Background()
	now := time.Now().Unix()

	// Insert an old metric (older than 3 seconds)
	err = s.SaveMetric(ctx, "old.metric", "string", 10, "stale_value", now-10)
	require.NoError(t, err)

	// Call the synchronous cleaner instead of waiting for the ticker
	err = s.cleanRetainedMetrics(ctx)
	require.NoError(t, err)

	// History should be empty for that metric
	hist, err := s.GetMetricHistory(ctx, "old.metric")
	require.NoError(t, err)
	require.Equal(t, "old.metric", hist.Name) // Metric definition should still exist
	require.Equal(t, 0, len(hist.History))    // But values should be gone
}

func TestSQLiteStorage_Ordering(t *testing.T) {
	s, err := NewSQLiteStorage(":memory:", 3600)
	require.NoError(t, err)
	defer func() {
		_ = s.Close()
	}()

	ctx := context.Background()

	// 1. Test Metric Ordering
	err = s.SaveMetric(ctx, "m1", "uint64", 1, "1", time.Now().Unix())
	require.NoError(t, err)
	err = s.SaveMetric(ctx, "m2", "uint64", 1, "2", time.Now().Unix())
	require.NoError(t, err)

	err = s.UpdateMetricOrder(ctx, "m1", 10)
	require.NoError(t, err)

	latest, err := s.GetLatestMetrics(ctx)
	require.NoError(t, err)

	var m1Found bool
	for _, m := range latest {
		if m.Name == "m1" {
			require.Equal(t, 10, m.DisplayOrder)
			m1Found = true
		} else if m.Name == "m2" {
			require.Equal(t, 0, m.DisplayOrder)
		}
	}
	require.True(t, m1Found)

	// 2. Test Panel Ordering
	err = s.UpdatePanelOrder(ctx, "VM1", 5)
	require.NoError(t, err)
	err = s.UpdatePanelOrder(ctx, "VM2", 1)
	require.NoError(t, err)

	configs, err := s.GetPanelsConfigs(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, len(configs))
	require.Equal(t, 5, configs["VM1"])
	require.Equal(t, 1, configs["VM2"])

	// Update existing panel
	err = s.UpdatePanelOrder(ctx, "VM1", 0)
	require.NoError(t, err)
	configs, err = s.GetPanelsConfigs(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, configs["VM1"])
}
