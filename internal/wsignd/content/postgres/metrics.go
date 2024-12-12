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
			WITH load_events AS (
				-- Get successful content load events with metrics
				SELECT
					EXTRACT(EPOCH FROM timestamp) AS event_ts,
					CASE
						WHEN metrics IS NOT NULL AND jsonb_typeof(metrics->>'loadTime') = 'number'
						THEN (metrics->>'loadTime')::float
						ELSE NULL
					END AS load_time,
					CASE
						WHEN metrics IS NOT NULL AND jsonb_typeof(metrics->>'renderTime') = 'number'
						THEN (metrics->>'renderTime')::float
						ELSE NULL
					END AS render_time
				FROM content_events
				WHERE 
					url = $1 
					AND timestamp >= $2
					AND type = 'CONTENT_LOADED'
			),
			error_events AS (
				-- Calculate error rates by code
				SELECT
					COALESCE(error->>'code', 'UNKNOWN_ERROR') AS error_code,
					COUNT(*) AS error_count
				FROM content_events
				WHERE 
					url = $1 
					AND timestamp >= $2
					AND type = 'CONTENT_ERROR'
					AND error IS NOT NULL
				GROUP BY error->>'code'
			)
			SELECT
				-- Base metrics
				COUNT(DISTINCT le.event_ts) AS load_count,
				COALESCE(SUM(ee.error_count), 0) AS error_count,
				COALESCE(MAX(le.event_ts), EXTRACT(EPOCH FROM $2)) AS last_seen_ts,
				
				-- Performance metrics with safe averages
				COALESCE(AVG(le.load_time) FILTER (WHERE le.load_time IS NOT NULL), 0) AS avg_load_time,
				COALESCE(AVG(le.render_time) FILTER (WHERE le.render_time IS NOT NULL), 0) AS avg_render_time,
				
				-- Error rates as JSONB object
				COALESCE(
					jsonb_object_agg(
						ee.error_code,
						ROUND(
							CAST(ee.error_count AS float) * 100.0 / 
							NULLIF(COUNT(DISTINCT le.event_ts) + SUM(ee.error_count), 0),
							2
						)
					) FILTER (WHERE ee.error_code IS NOT NULL),
					'{}'::jsonb
				) AS error_rates
			FROM load_events le
			FULL OUTER JOIN error_events ee ON true;
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
			// No events found - return empty metrics
			metrics.LastSeen = int64(since.Unix())
			return nil
		}
		if err != nil {
			return err
		}

		// Parse error rates from JSONB
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
