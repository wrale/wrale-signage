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
		// Use CTEs for clear and efficient metric calculation
		const query = `
			WITH base_stats AS (
				SELECT
					COUNT(*) FILTER (WHERE type = 'CONTENT_LOADED') AS load_count,
					COUNT(*) FILTER (WHERE type = 'CONTENT_ERROR') AS error_count,
					MAX(EXTRACT(EPOCH FROM timestamp)) AS last_seen_ts
				FROM content_events
				WHERE url = $1 AND timestamp >= $2
			),
			performance_stats AS (
				SELECT
					COALESCE(AVG((metrics->>'loadTime')::float), 0) AS avg_load_time,
					COALESCE(AVG((metrics->>'renderTime')::float), 0) AS avg_render_time
				FROM content_events
				WHERE 
					url = $1 
					AND timestamp >= $2
					AND type = 'CONTENT_LOADED'
					AND metrics->>'loadTime' IS NOT NULL
			),
			error_stats AS (
				SELECT
					error->>'code' AS error_code,
					COUNT(*) AS code_count,
					-- Calculate error rate against total number of loads and errors
					ROUND(
						COUNT(*) * 100.0 / NULLIF(
							(SELECT load_count + error_count FROM base_stats),
							0
						),
						2
					) AS error_rate
				FROM content_events
				WHERE 
					url = $1 
					AND timestamp >= $2
					AND type = 'CONTENT_ERROR'
					AND error->>'code' IS NOT NULL
				GROUP BY error->>'code'
			)
			SELECT
				b.load_count,
				b.error_count,
				b.last_seen_ts,
				p.avg_load_time,
				p.avg_render_time,
				COALESCE(
					jsonb_object_agg(
						e.error_code,
						e.error_rate
					) FILTER (WHERE e.error_code IS NOT NULL),
					'{}'::jsonb
				) AS error_rates
			FROM base_stats b
			CROSS JOIN performance_stats p
			LEFT JOIN error_stats e ON true;
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
