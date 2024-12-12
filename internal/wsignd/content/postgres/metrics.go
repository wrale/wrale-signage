package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
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
		// First get basic metrics
		const baseQuery = `
			SELECT
				COUNT(*) FILTER (WHERE type = 'CONTENT_LOADED') AS load_count,
				COUNT(*) FILTER (WHERE type = 'CONTENT_ERROR') AS error_count,
				COALESCE(MAX(EXTRACT(EPOCH FROM timestamp)), EXTRACT(EPOCH FROM $2)) AS last_seen_ts,
				COALESCE(
					AVG((metrics->>'loadTime')::float) FILTER (
						WHERE type = 'CONTENT_LOADED' 
						AND metrics->>'loadTime' IS NOT NULL
					),
					0
				) AS avg_load_time,
				COALESCE(
					AVG((metrics->>'renderTime')::float) FILTER (
						WHERE type = 'CONTENT_LOADED'
						AND metrics->>'renderTime' IS NOT NULL
					),
					0
				) AS avg_render_time
			FROM content_events
			WHERE url = $1 AND timestamp >= $2;
		`

		err := tx.QueryRowContext(ctx, baseQuery, url, since).Scan(
			&metrics.LoadCount,
			&metrics.ErrorCount,
			&metrics.LastSeen,
			&metrics.AvgLoadTime,
			&metrics.AvgRenderTime,
		)
		if err != nil {
			return err
		}

		// Then get error rates if we have any errors
		if metrics.ErrorCount > 0 {
			const errorQuery = `
				WITH total_events AS (
					SELECT
						COUNT(*) AS total
					FROM content_events
					WHERE url = $1 AND timestamp >= $2
				)
				SELECT
					COALESCE(
						jsonb_object_agg(
							error->>'code',
							ROUND(
								COUNT(*)::float * 100.0 / NULLIF((SELECT total FROM total_events), 0),
								2
							)::float
						) FILTER (WHERE error->>'code' IS NOT NULL),
						'{}'::jsonb
					) AS error_rates
				FROM content_events
				WHERE 
					url = $1 
					AND timestamp >= $2 
					AND type = 'CONTENT_ERROR';
			`

			var errorRatesJSON []byte
			if err := tx.QueryRowContext(ctx, errorQuery, url, since).Scan(&errorRatesJSON); err != nil {
				return err
			}

			if len(errorRatesJSON) > 0 {
				if err := json.Unmarshal(errorRatesJSON, &metrics.ErrorRates); err != nil {
					return err
				}
			}
		}

		return nil
	})

	if err != nil {
		if err == sql.ErrNoRows {
			// No events found - return empty metrics with since timestamp
			metrics.LastSeen = int64(since.Unix())
			return metrics, nil
		}
		return nil, database.MapError(err, op)
	}

	return metrics, nil
}
