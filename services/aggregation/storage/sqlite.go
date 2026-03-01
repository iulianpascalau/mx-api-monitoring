package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/iulianpascalau/api-monitoring/services/aggregation/common"
	_ "github.com/mattn/go-sqlite3"
	logger "github.com/multiversx/mx-chain-logger-go"
)

var log = logger.GetOrCreate("storage")

// sqliteStorage is the sqlite implementation for metrics storage
type sqliteStorage struct {
	db               *sql.DB
	retentionSeconds int
	cancelFunc       context.CancelFunc
	wg               sync.WaitGroup
}

// NewSQLiteStorage creates the database, schema, and starts the retention cleaner
func NewSQLiteStorage(dbPath string, retentionSeconds int) (*sqliteStorage, error) {
	err := prepareDirectories(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create initial empty DB file: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	err = createSchema(db)
	if err != nil {
		_ = db.Close()
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	s := &sqliteStorage{
		db:               db,
		retentionSeconds: retentionSeconds,
		cancelFunc:       cancel,
	}

	s.startRetentionCleaner(ctx)

	return s, nil
}

func prepareDirectories(dbPath string) error {
	return os.MkdirAll(filepath.Dir(dbPath), os.ModePerm)
}

// CleanRetainedMetrics executes the retention cleanup query synchronously.
func (s *sqliteStorage) cleanRetainedMetrics(ctx context.Context) error {
	nowSec := time.Now().Unix()
	cutoff := nowSec - int64(s.retentionSeconds)
	_, err := s.db.ExecContext(ctx, "DELETE FROM metrics_values WHERE recorded_at < ?", cutoff)
	return err
}

func createSchema(db *sql.DB) error {

	schema := `
	CREATE TABLE IF NOT EXISTS metrics (
		name            TEXT    NOT NULL PRIMARY KEY,
		type            TEXT    NOT NULL,
		num_aggregation INTEGER NOT NULL DEFAULT 1,
		display_order   INTEGER NOT NULL DEFAULT 0
	);

	CREATE TABLE IF NOT EXISTS panel_configs (
		name            TEXT    NOT NULL PRIMARY KEY,
		display_order   INTEGER NOT NULL DEFAULT 0
	);

	CREATE TABLE IF NOT EXISTS metrics_values (
		metric_name TEXT    NOT NULL REFERENCES metrics(name) ON DELETE CASCADE,
		value       TEXT    NOT NULL,
		recorded_at INTEGER NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_metrics_values_name ON metrics_values(metric_name);
	CREATE INDEX IF NOT EXISTS idx_metrics_values_recorded_at ON metrics_values(recorded_at);
	`

	_, err := db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	// Migration: ensure display_order column exists in metrics
	_, _ = db.Exec("ALTER TABLE metrics ADD COLUMN display_order INTEGER NOT NULL DEFAULT 0;")

	// Make sure ON DELETE CASCADE works if enabled globally
	_, _ = db.Exec("PRAGMA foreign_keys = ON;")

	return nil
}

// SaveMetric upserts the metric definition, inserts the value, and prunes old entries based on numAggregation
func (s *sqliteStorage) SaveMetric(ctx context.Context, name string, metricType string, numAggregation int, valString string, recordedAt int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO metrics (name, type, num_aggregation) 
		VALUES (?, ?, ?) 
		ON CONFLICT(name) DO UPDATE SET 
			type=excluded.type, 
			num_aggregation=excluded.num_aggregation
	`, name, metricType, numAggregation)
	if err != nil {
		return fmt.Errorf("failed to upsert metric definition: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO metrics_values (metric_name, value, recorded_at)
		VALUES (?, ?, ?)
	`, name, valString, recordedAt)
	if err != nil {
		return fmt.Errorf("failed to insert metric value: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		DELETE FROM metrics_values
		WHERE metric_name = ?
		  AND rowid NOT IN (
			  SELECT rowid FROM metrics_values
			  WHERE metric_name = ?
			  ORDER BY recorded_at DESC
			  LIMIT ?
		  )
	`, name, name, numAggregation)
	if err != nil {
		return fmt.Errorf("failed to trim metric aggregation window: %w", err)
	}

	return tx.Commit()
}

// GetLatestMetrics fetches the most recent value for each metric
func (s *sqliteStorage) GetLatestMetrics(ctx context.Context) ([]common.MetricHistory, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT m.name, m.type, m.num_aggregation, m.display_order, v.value, v.recorded_at
		FROM metrics m
		LEFT JOIN (
			SELECT metric_name, value, recorded_at,
				ROW_NUMBER() OVER(PARTITION BY metric_name ORDER BY recorded_at DESC) as rn
			FROM metrics_values
		) v ON m.name = v.metric_name AND v.rn = 1
	`)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var results []common.MetricHistory

	for rows.Next() {
		var h common.MetricHistory
		var val string
		var recAt int64

		err = rows.Scan(&h.Name, &h.Type, &h.NumAggregation, &h.DisplayOrder, &val, &recAt)
		if err != nil {
			return nil, err
		}

		h.History = []common.MetricValue{
			{
				Value:      val,
				RecordedAt: recAt,
			},
		}
		results = append(results, h)
	}

	return results, rows.Err()
}

// GetMetricHistory returns the metric configuration and up to 'num_aggregation' historical values
func (s *sqliteStorage) GetMetricHistory(ctx context.Context, name string) (*common.MetricHistory, error) {
	var h common.MetricHistory

	err := s.db.QueryRowContext(ctx, "SELECT name, type, num_aggregation, display_order FROM metrics WHERE name = ?", name).Scan(&h.Name, &h.Type, &h.NumAggregation, &h.DisplayOrder)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("metric not found")
	}
	if err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT value, recorded_at 
		FROM metrics_values 
		WHERE metric_name = ? 
		ORDER BY recorded_at
	`, name)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	for rows.Next() {
		var val string
		var recAt int64

		err = rows.Scan(&val, &recAt)
		if err != nil {
			return nil, err
		}

		h.History = append(h.History, common.MetricValue{Value: val, RecordedAt: recAt})
	}

	return &h, rows.Err()
}

// DeleteMetric forcefully deletes a metric and all its values from the database
func (s *sqliteStorage) DeleteMetric(ctx context.Context, name string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM metrics WHERE name = ?", name)
	return err
}

// UpdateMetricOrder updates the display order of a specific metric
func (s *sqliteStorage) UpdateMetricOrder(ctx context.Context, name string, order int) error {
	_, err := s.db.ExecContext(ctx, "UPDATE metrics SET display_order = ? WHERE name = ?", order, name)
	return err
}

// UpdatePanelOrder updates the display order of a specific panel (VM)
func (s *sqliteStorage) UpdatePanelOrder(ctx context.Context, name string, order int) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO panel_configs (name, display_order) 
		VALUES (?, ?) 
		ON CONFLICT(name) DO UPDATE SET display_order=excluded.display_order
	`, name, order)
	return err
}

// GetPanelsConfigs returns the display configurations for all panels
func (s *sqliteStorage) GetPanelsConfigs(ctx context.Context) (map[string]int, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT name, display_order FROM panel_configs")
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	res := make(map[string]int)
	for rows.Next() {
		var name string
		var order int
		if err := rows.Scan(&name, &order); err != nil {
			return nil, err
		}
		res[name] = order
	}
	return res, rows.Err()
}

func (s *sqliteStorage) startRetentionCleaner(ctx context.Context) {
	s.wg.Add(1)

	// max(RetentionSeconds/10, 60)
	intervalSec := s.retentionSeconds / 10
	if intervalSec < 60 {
		intervalSec = 60
	}

	ticker := time.NewTicker(time.Duration(intervalSec) * time.Second)

	go func() {
		defer s.wg.Done()
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				log.Debug("running retention cleanup")

				err := s.cleanRetainedMetrics(ctx)
				if err != nil {
					log.Warn("failed to cleanup retained metrics", "error", err)
				}
			}
		}
	}()
}

// Close closes the database and stops background routines
func (s *sqliteStorage) Close() error {
	s.cancelFunc()
	s.wg.Wait()
	return s.db.Close()
}

// IsInterfaceNil returns true if the value under the interface is nil
func (s *sqliteStorage) IsInterfaceNil() bool {
	return s == nil
}
