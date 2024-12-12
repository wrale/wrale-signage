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
		const query = `
			WITH stats AS (
				SELECT
					-- Total event counts
					COUNT(*) FILTER (WHERE type = 'CONTENT_LOADED') AS load_count,
					COUNT(*) FILTER (WHERE type = 'CONTENT_ERROR') AS error_count,
					-- Latest timestamp
					MAX(EXTRACT(EPOCH FROM timestamp)) AS last_seen_ts,
					-- Performance metrics (only from load events)
					AVG((metrics->>'loadTime')::float) FILTER (
						WHERE type = 'CONTENT_LOADED' 
						AND metrics->>'loadTime' IS NOT NULL
					) AS avg_load_time,
					AVG((metrics->>'renderTime')::float) FILTER (
						WHERE type = 'CONTENT_LOADED'
						AND metrics->>'renderTime' IS NOT NULL
					) AS avg_render_time
				FROM content_events
				WHERE url = $1 AND timestamp >= $2
			),
			error_counts AS (
				SELECT 
					error->>'code' AS error_code,
					COUNT(*) AS count,
					COUNT(*) * 100.0 / NULLIF(
						(SELECT load_count + error_count FROM stats),
						0
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
				COALESCE(s.load_count, 0),
				COALESCE(s.error_count, 0),
				COALESCE(s.last_seen_ts, EXTRACT(EPOCH FROM $2)),
				COALESCE(s.avg_load_time, 0),
				COALESCE(s.avg_render_time, 0),
				COALESCE(
					jsonb_object_agg(
						e.error_code,
						ROUND(e.error_rate::numeric, 2)
					) FILTER (WHERE e.error_code IS NOT NULL),
					'{}'::jsonb
				) AS error_rates
			FROM stats s
			LEFT JOIN error_counts e ON true;
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
			// No events found - return empty metrics with since timestamp
			metrics.LastSeen = int64(since.Unix())
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
