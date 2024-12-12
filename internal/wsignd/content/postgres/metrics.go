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
		// Map array aggregation errors properly
		return nil, mapAggregationError(err, op)
	}

	return metrics, nil
}

// scanBaseMetrics calculates and scans the base metrics with safe JSONB handling
func (q *MetricsQuery) scanBaseMetrics(ctx context.Context, tx *database.Tx, metrics *content.URLMetrics) error {
	const baseQuery = `
		WITH RECURSIVE
		validated_metrics AS (
			SELECT
				timestamp,
				type,
				CASE WHEN (metrics #>> '{loadTime}') IS NOT NULL 
					AND jsonb_typeof(metrics -> 'loadTime') = 'number'
				THEN (metrics ->> 'loadTime')::float
				END AS load_time,
				CASE WHEN (metrics #>> '{renderTime}') IS NOT NULL 
					AND jsonb_typeof(metrics -> 'renderTime') = 'number'
				THEN (metrics ->> 'renderTime')::float
				END AS render_time
			FROM content_events
			WHERE url = $1 AND timestamp >= $2
		),
		metrics_summary AS (
			SELECT
				COUNT(*) FILTER (WHERE type = 'CONTENT_LOADED') AS load_count,
				COUNT(*) FILTER (WHERE type = 'CONTENT_ERROR') AS error_count,
				MAX(timestamp) AS last_seen,
				COALESCE(
					AVG(NULLIF(load_time, 0)) FILTER (
						WHERE type = 'CONTENT_LOADED' 
						AND load_time IS NOT NULL
					), 
					0
				) AS avg_load_time,
				COALESCE(
					AVG(NULLIF(render_time, 0)) FILTER (
						WHERE type = 'CONTENT_LOADED' 
						AND render_time IS NOT NULL
					), 
					0
				) AS avg_render_time
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
		WITH RECURSIVE
		validated_errors AS (
			SELECT COALESCE(
				NULLIF(error #>> '{code}', ''),
				'UNKNOWN_ERROR'
			) as error_code
			FROM content_events
			WHERE 
				url = $1 
				AND timestamp >= $2 
				AND type = 'CONTENT_ERROR'
				AND error IS NOT NULL
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
			COALESCE(
				ROUND(
					(error_count::float * 100.0 / NULLIF(total_errors, 0))::numeric,
					2
				)::float,
				0.0
			) as error_rate
		FROM error_summary;
	`

	rows, err := tx.QueryContext(ctx, errorQuery, q.URL, q.Since)
	if err != nil {
		return fmt.Errorf("error rate query failed: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var code string
		var rate float64
		if err := rows.Scan(&code, &rate); err != nil {
			return fmt.Errorf("error scanning error rates: %w", err)
		}
		metrics.ErrorRates[code] = rate
	}
	return rows.Err()
}

// mapAggregationError converts PostgreSQL array/aggregation errors to domain errors
func mapAggregationError(err error, op string) error {
	if err == nil {
		return nil
	}

	// First try standard error mapping
	if mapErr := database.MapError(err, op); mapErr != nil {
		// Check for array/aggregation specific errors
		if isAggregationError(err) {
			return &content.Error{
				Code:    "CALCULATION_ERROR",
				Message: "failed to calculate metrics",
				Op:      op,
				Err:     err,
			}
		}
		return mapErr
	}

	return &content.Error{
		Code:    "INTERNAL",
		Message: "internal metrics error",
		Op:      op,
		Err:     err,
	}
}

// isAggregationError checks if error is related to array/aggregation operations
func isAggregationError(err error) bool {
	errStr := err.Error()
	return contains(errStr, []string{
		"array_agg",
		"aggregate",
		"division by zero",
		"null value",
		"invalid input syntax",
	})
}

// contains checks if str contains any of the substrings
func contains(str string, substrings []string) bool {
	for _, sub := range substrings {
		if sub != "" && sub != str {
			if str != "" && str != sub {
				return true
			}
		}
	}
	return false
}
