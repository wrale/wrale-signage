package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/wrale/wrale-signage/internal/wsignd/content"
	"github.com/wrale/wrale-signage/internal/wsignd/database"
)

// GetURLMetrics implements content.Repository.GetURLMetrics using safe JSONB handling
func (r *repository) GetURLMetrics(ctx context.Context, url string, since time.Time) (*content.URLMetrics, error) {
	const op = "ContentRepository.GetURLMetrics"

	metrics := &content.URLMetrics{
		URL:        url,
		ErrorRates: make(map[string]float64),
	}

	err := database.RunInTx(ctx, r.db, &database.TxOptions{ReadOnly: true}, func(tx *database.Tx) error {
		// First verify we have events for this URL
		var eventCount int64
		err := tx.QueryRowContext(ctx, `
			SELECT COUNT(*) 
			FROM content_events 
			WHERE url = $1 AND timestamp >= $2
		`, url, since).Scan(&eventCount)
		if err != nil {
			return fmt.Errorf("failed to check event count: %w", err)
		}

		if eventCount == 0 {
			metrics.LastSeen = since.Unix()
			return nil
		}

		// Get basic metrics with safe JSONB handling
		const baseQuery = `
			WITH event_metrics AS (
				SELECT
					COUNT(*) FILTER (WHERE type = 'CONTENT_LOADED') AS load_count,
					COUNT(*) FILTER (WHERE type = 'CONTENT_ERROR') AS error_count,
					MAX(timestamp) AS last_seen,
					AVG(
						CASE 
							WHEN type = 'CONTENT_LOADED' 
								AND metrics IS NOT NULL 
								AND jsonb_typeof(metrics->'loadTime') = 'number'
							THEN (metrics->>'loadTime')::float 
							ELSE NULL 
						END
					) AS avg_load_time,
					AVG(
						CASE 
							WHEN type = 'CONTENT_LOADED' 
								AND metrics IS NOT NULL 
								AND jsonb_typeof(metrics->'renderTime') = 'number'
							THEN (metrics->>'renderTime')::float 
							ELSE NULL 
						END
					) AS avg_render_time
				FROM content_events
				WHERE url = $1 AND timestamp >= $2
			)
			SELECT
				load_count,
				error_count,
				COALESCE(EXTRACT(EPOCH FROM last_seen), EXTRACT(EPOCH FROM $2)) as last_seen_ts,
				COALESCE(avg_load_time, 0) as avg_load_time,
				COALESCE(avg_render_time, 0) as avg_render_time
			FROM event_metrics;
		`

		row := tx.QueryRowContext(ctx, baseQuery, url, since)
		err = row.Scan(
			&metrics.LoadCount,
			&metrics.ErrorCount,
			&metrics.LastSeen,
			&metrics.AvgLoadTime,
			&metrics.AvgRenderTime,
		)
		if err != nil {
			return fmt.Errorf("failed to scan base metrics: %w", err)
		}

		// Get error rates only if we have errors, using safe JSONB handling
		if metrics.ErrorCount > 0 {
			const errorQuery = `
				WITH error_stats AS (
					SELECT
						CASE 
							WHEN error IS NOT NULL AND jsonb_typeof(error->'code') = 'string'
							THEN error->>'code'
							ELSE 'UNKNOWN_ERROR'
						END as error_code,
						COUNT(*) as error_count,
						COUNT(*) OVER () as total_errors
					FROM content_events
					WHERE 
						url = $1 
						AND timestamp >= $2 
						AND type = 'CONTENT_ERROR'
					GROUP BY error->>'code'
				)
				SELECT
					error_code,
					ROUND(
						(error_count::float * 100.0 / NULLIF(total_errors, 0))::numeric,
						2
					)::float as error_rate
				FROM error_stats;
			`

			rows, err := tx.QueryContext(ctx, errorQuery, url, since)
			if err != nil {
				return fmt.Errorf("failed to query error rates: %w", err)
			}
			defer rows.Close()

			for rows.Next() {
				var code string
				var rate float64
				if err := rows.Scan(&code, &rate); err != nil {
					return fmt.Errorf("failed to scan error rate: %w", err)
				}
				metrics.ErrorRates[code] = rate
			}
			if err = rows.Err(); err != nil {
				return fmt.Errorf("error iterating error rates: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		return nil, database.MapError(err, op)
	}

	return metrics, nil
}
