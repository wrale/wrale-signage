package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/wrale/wrale-signage/internal/wsignd/content"
	"github.com/wrale/wrale-signage/internal/wsignd/database"
)

// GetURLMetrics implements content.Repository.GetURLMetrics
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
			return fmt.Errorf("failed to check event existence: %w", err)
		}

		if eventCount == 0 {
			metrics.LastSeen = since.Unix()
			return nil
		}

		// Get basic metrics with enhanced error handling
		const baseQuery = `
			WITH event_metrics AS (
				SELECT
					COUNT(*) FILTER (WHERE type = 'CONTENT_LOADED') AS load_count,
					COUNT(*) FILTER (WHERE type = 'CONTENT_ERROR') AS error_count,
					MAX(timestamp) AS last_seen,
					AVG(CASE 
						WHEN type = 'CONTENT_LOADED' AND metrics->>'loadTime' IS NOT NULL 
						THEN (metrics->>'loadTime')::float 
						ELSE NULL 
					END) AS avg_load_time,
					AVG(CASE 
						WHEN type = 'CONTENT_LOADED' AND metrics->>'renderTime' IS NOT NULL 
						THEN (metrics->>'renderTime')::float 
						ELSE NULL 
					END) AS avg_render_time
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

		// Get error rates only if we have errors
		if metrics.ErrorCount > 0 {
			var errorRatesJSON []byte
			err = tx.QueryRowContext(ctx, `
				WITH error_stats AS (
					SELECT
						error->>'code' as error_code,
						COUNT(*) as error_count,
						(SELECT COUNT(*) FROM content_events WHERE url = $1 AND timestamp >= $2) as total_events
					FROM content_events
					WHERE 
						url = $1 
						AND timestamp >= $2 
						AND type = 'CONTENT_ERROR'
						AND error->>'code' IS NOT NULL
					GROUP BY error->>'code'
				)
				SELECT
					COALESCE(
						jsonb_object_agg(
							error_code,
							ROUND(
								(error_count::float * 100.0 / total_events::float)::numeric,
								2
							)::float
						),
						'{}'::jsonb
					) as error_rates
				FROM error_stats;
			`, url, since).Scan(&errorRatesJSON)

			if err != nil && err != sql.ErrNoRows {
				return fmt.Errorf("failed to scan error rates: %w", err)
			}

			if len(errorRatesJSON) > 0 {
				if err := json.Unmarshal(errorRatesJSON, &metrics.ErrorRates); err != nil {
					return fmt.Errorf("failed to unmarshal error rates: %w", err)
				}
			}
		}

		return nil
	})

	if err != nil {
		return nil, database.MapError(err, op)
	}

	return metrics, nil
}
