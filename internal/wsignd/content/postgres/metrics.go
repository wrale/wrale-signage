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
		// Use a CTE to calculate base metrics
		const query = `
			WITH event_stats AS (
				SELECT
					COUNT(*) AS total_events,
					COUNT(*) FILTER (WHERE type = 'CONTENT_ERROR') AS error_count,
					MAX(EXTRACT(EPOCH FROM timestamp)) AS last_seen_ts,
					AVG(CASE
						WHEN metrics->>'loadTime' IS NOT NULL 
						THEN (metrics->>'loadTime')::float 
						ELSE 0 
					END) AS avg_load_time,
					AVG(CASE
						WHEN metrics->>'renderTime' IS NOT NULL 
						THEN (metrics->>'renderTime')::float 
						ELSE 0 
					END) AS avg_render_time
				FROM content_events
				WHERE url = $1 AND timestamp >= $2
			),
			error_stats AS (
				SELECT
					(error->>'code')::text AS error_code,
					COUNT(*) AS code_count,
					COUNT(*) * 100.0 / NULLIF((SELECT total_events FROM event_stats), 0) AS error_rate
				FROM content_events
				WHERE 
					url = $1 
					AND timestamp >= $2
					AND type = 'CONTENT_ERROR'
					AND error->>'code' IS NOT NULL
				GROUP BY error->>'code'
			)
			SELECT
				es.total_events,
				es.error_count,
				es.last_seen_ts,
				es.avg_load_time,
				es.avg_render_time,
				COALESCE(
					json_object_agg(
						er.error_code,
						er.error_rate
					) FILTER (WHERE er.error_code IS NOT NULL),
					'{}'::json
				) AS error_rates
			FROM event_stats es
			LEFT JOIN error_stats er ON true;
		`

		var errorRatesJSON []byte
		err := tx.QueryRowContext(ctx, query, url, since).Scan(
			&metrics.LoadCount,
			&metrics.ErrorCount,
			&metrics.LastSeen,
			&metrics.AvgLoadTime,
			&metrics.AvgRenderTime,
			&errorRatesJSON,
		)

		if err == sql.ErrNoRows {
			// No events found, return empty metrics
			return nil
		}
		if err != nil {
			return err
		}

		// Parse error rates
		if len(errorRatesJSON) > 0 {
			if err := json.Unmarshal(errorRatesJSON, &metrics.ErrorRates); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, database.MapError(err, op)
	}

	return metrics, nil
}
