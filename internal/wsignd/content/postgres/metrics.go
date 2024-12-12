package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/wrale/wrale-signage/internal/wsignd/content"
	"github.com/wrale/wrale-signage/internal/wsignd/database"
)

// MetricsQuery encapsulates SQL for metrics calculation
type MetricsQuery struct {
	URL   string
	Since time.Time
}

// GetURLMetrics implements content.Repository.GetURLMetrics using safe JSONB handling
func (r *repository) GetURLMetrics(ctx context.Context, url string, since time.Time) (*content.URLMetrics, error) {
	const op = "ContentRepository.GetURLMetrics"

	metrics := &content.URLMetrics{
		URL:        url,
		ErrorRates: make(map[string]float64),
	}

	// Use serializable isolation for metrics calculations
	err := database.RunInTx(ctx, r.db, &database.TxOptions{
		ReadOnly:  true,
		Isolation: database.LevelSerializable,
	}, func(tx *database.Tx) error {
		q := MetricsQuery{URL: url, Since: since}

		// First verify we have events
		var count int64
		err := tx.QueryRowContext(ctx, `
			SELECT COUNT(*) 
			FROM content_events 
			WHERE url = $1 AND timestamp >= $2
		`, q.URL, q.Since).Scan(&count)
		if err != nil {
			return fmt.Errorf("failed to count events: %w", err)
		}

		if count == 0 {
			metrics.LastSeen = since.Unix()
			return nil
		}

		// Calculate base metrics first
		if err := q.scanBaseMetrics(ctx, tx, metrics); err != nil {
			return fmt.Errorf("failed to get base metrics: %w", err)
		}

		// Only calculate error rates if we have errors
		if metrics.ErrorCount > 0 {
			if err := q.scanErrorRates(ctx, tx, metrics); err != nil {
				return fmt.Errorf("failed to get error rates: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		return nil, mapPostgresError(err, op)
	}

	return metrics, nil
}

// scanBaseMetrics calculates and scans the base metrics with safe JSONB handling
func (q *MetricsQuery) scanBaseMetrics(ctx context.Context, tx *database.Tx, metrics *content.URLMetrics) error {
	const baseQuery = `
		WITH validated_metrics AS (
			SELECT
				timestamp,
				type,
				CASE
					WHEN jsonb_typeof(metrics) = 'object'
					AND jsonb_typeof(metrics->'loadTime') = 'number'
					THEN (metrics->>'loadTime')::float
				END AS load_time,
				CASE
					WHEN jsonb_typeof(metrics) = 'object'
					AND jsonb_typeof(metrics->'renderTime') = 'number'
					THEN (metrics->>'renderTime')::float
				END AS render_time
			FROM content_events
			WHERE url = $1 AND timestamp >= $2
		),
		metrics_summary AS (
			SELECT
				COUNT(*) FILTER (WHERE type = 'CONTENT_LOADED') AS load_count,
				COUNT(*) FILTER (WHERE type = 'CONTENT_ERROR') AS error_count,
				MAX(timestamp) AS last_seen,
				AVG(load_time) FILTER (WHERE type = 'CONTENT_LOADED' AND load_time IS NOT NULL) AS avg_load_time,
				AVG(render_time) FILTER (WHERE type = 'CONTENT_LOADED' AND render_time IS NOT NULL) AS avg_render_time
			FROM validated_metrics
		)
		SELECT
			COALESCE(load_count, 0),
			COALESCE(error_count, 0),
			COALESCE(EXTRACT(EPOCH FROM last_seen), EXTRACT(EPOCH FROM $2)) as last_seen_ts,
			COALESCE(avg_load_time, 0),
			COALESCE(avg_render_time, 0)
		FROM metrics_summary;
	`

	return tx.QueryRowContext(ctx, baseQuery, q.URL, q.Since).Scan(
		&metrics.LoadCount,
		&metrics.ErrorCount,
		&metrics.LastSeen,
		&metrics.AvgLoadTime,
		&metrics.AvgRenderTime,
	)
}

// scanErrorRates calculates and scans error rates with safe JSONB handling
func (q *MetricsQuery) scanErrorRates(ctx context.Context, tx *database.Tx, metrics *content.URLMetrics) error {
	const errorQuery = `
		WITH validated_errors AS (
			SELECT
				CASE 
					WHEN jsonb_typeof(error) = 'object'
					AND jsonb_typeof(error->'code') = 'string'
					THEN error->>'code'
					ELSE 'UNKNOWN_ERROR'
				END as error_code
			FROM content_events
			WHERE 
				url = $1 
				AND timestamp >= $2 
				AND type = 'CONTENT_ERROR'
		),
		error_summary AS (
			SELECT
				error_code,
				COUNT(*) as error_count,
				COUNT(*) OVER () as total_errors
			FROM validated_errors
			GROUP BY error_code
		)
		SELECT
			error_code,
			ROUND(
				(error_count::float * 100.0 / NULLIF(total_errors, 0))::numeric,
				2
			)::float as error_rate
		FROM error_summary;
	`

	rows, err := tx.QueryContext(ctx, errorQuery, q.URL, q.Since)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var code string
		var rate float64
		if err := rows.Scan(&code, &rate); err != nil {
			return err
		}
		metrics.ErrorRates[code] = rate
	}
	return rows.Err()
}
