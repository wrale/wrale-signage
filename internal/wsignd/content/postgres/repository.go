package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/wrale/wrale-signage/internal/wsignd/content"
	"github.com/wrale/wrale-signage/internal/wsignd/database"
)

type repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *repository {
	return &repository{db: db}
}

func (r *repository) SaveEvent(ctx context.Context, event content.Event) error {
	const op = "ContentRepository.SaveEvent"

	var metrics map[string]interface{}
	if event.Metrics != nil {
		metrics = map[string]interface{}{
			"loadTime":        event.Metrics.LoadTime,
			"renderTime":      event.Metrics.RenderTime,
			"interactiveTime": event.Metrics.InteractiveTime,
		}
		if event.Metrics.ResourceStats != nil {
			metrics["resourceStats"] = event.Metrics.ResourceStats
		}
	}
	metricsJSON, err := json.Marshal(metrics)
	if err != nil {
		return database.MapError(err, op)
	}

	var errorData map[string]interface{}
	if event.Error != nil {
		errorData = map[string]interface{}{
			"code":    event.Error.Code,
			"message": event.Error.Message,
		}
		if event.Error.Details != nil {
			errorData["details"] = event.Error.Details
		}
	}
	errorJSON, err := json.Marshal(errorData)
	if err != nil {
		return database.MapError(err, op)
	}

	contextJSON, err := json.Marshal(event.Context)
	if err != nil {
		return database.MapError(err, op)
	}

	err = database.RunInTx(ctx, r.db, nil, func(tx *database.Tx) error {
		// Verify display exists
		var exists bool
		err := tx.QueryRowContext(ctx,
			"SELECT EXISTS(SELECT 1 FROM displays WHERE id = $1)",
			event.DisplayID,
		).Scan(&exists)
		if err != nil {
			return err
		}
		if !exists {
			return database.MapError(sql.ErrNoRows, op)
		}

		// Insert event
		_, err = tx.ExecContext(ctx, `
			INSERT INTO content_events (
				id, display_id, type, url, timestamp,
				error, metrics, context
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`,
			event.ID,
			event.DisplayID,
			event.Type,
			event.URL,
			event.Timestamp,
			errorJSON,
			metricsJSON,
			contextJSON,
		)
		return err
	})

	if err != nil {
		return database.MapError(err, op)
	}

	return nil
}

func (r *repository) GetURLMetrics(ctx context.Context, url string, since time.Time) (*content.URLMetrics, error) {
	const op = "ContentRepository.GetURLMetrics"

	var metrics content.URLMetrics
	metrics.URL = url
	metrics.ErrorRates = make(map[string]float64)

	err := database.RunInTx(ctx, r.db, &database.TxOptions{ReadOnly: true}, func(tx *database.Tx) error {
		// Get load and error counts
		err := tx.QueryRowContext(ctx, `
			SELECT 
				COUNT(*) FILTER (WHERE type = 'CONTENT_LOADED'),
				COUNT(*) FILTER (WHERE type = 'CONTENT_ERROR')
			FROM content_events 
			WHERE url = $1 AND timestamp >= $2
		`, url, since).Scan(&metrics.LoadCount, &metrics.ErrorCount)
		if err != nil {
			return err
		}

		// If no events found, return empty metrics
		if metrics.LoadCount == 0 && metrics.ErrorCount == 0 {
			return nil
		}

		// Get last seen timestamp
		err = tx.QueryRowContext(ctx, `
			SELECT EXTRACT(EPOCH FROM MAX(timestamp))::bigint
			FROM content_events 
			WHERE url = $1
		`, url).Scan(&metrics.LastSeen)
		if err != nil && err != sql.ErrNoRows {
			return err
		}

		// Get average timing metrics
		err = tx.QueryRowContext(ctx, `
			WITH valid_metrics AS (
				SELECT 
					(metrics->>'loadTime')::float8 as load_time,
					(metrics->>'renderTime')::float8 as render_time
				FROM content_events 
				WHERE url = $1 
					AND timestamp >= $2
					AND type = 'CONTENT_LOADED'
					AND metrics IS NOT NULL
					AND metrics ? 'loadTime' 
					AND metrics ? 'renderTime'
					AND jsonb_typeof(metrics->'loadTime') = 'number'
					AND jsonb_typeof(metrics->'renderTime') = 'number'
			)
			SELECT 
				COALESCE(AVG(load_time), 0),
				COALESCE(AVG(render_time), 0)
			FROM valid_metrics
		`, url, since).Scan(&metrics.AvgLoadTime, &metrics.AvgRenderTime)
		if err != nil && err != sql.ErrNoRows {
			return err
		}

		// Get error rates
		rows, err := tx.QueryContext(ctx, `
			WITH error_counts AS (
				SELECT 
					error->>'code' as error_code,
					COUNT(*) as code_count
				FROM content_events
				WHERE url = $1 
					AND timestamp >= $2
					AND type = 'CONTENT_ERROR'
					AND error IS NOT NULL
					AND error ? 'code'
					AND jsonb_typeof(error->'code') = 'string'
				GROUP BY error->>'code'
			),
			total AS (
				SELECT COUNT(*)::float8 as total_count
				FROM content_events 
				WHERE url = $1 AND timestamp >= $2
			)
			SELECT 
				error_code,
				code_count / NULLIF(total_count, 0)
			FROM error_counts, total
		`, url, since)
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
	})

	if err != nil {
		return nil, database.MapError(err, op)
	}

	return &metrics, nil
}

func (r *repository) GetDisplayEvents(ctx context.Context, displayID uuid.UUID, since time.Time) ([]content.Event, error) {
	const op = "ContentRepository.GetDisplayEvents"

	var events []content.Event

	err := database.RunInTx(ctx, r.db, &database.TxOptions{ReadOnly: true}, func(tx *database.Tx) error {
		rows, err := tx.QueryContext(ctx, `
			SELECT 
				id, display_id, type, url, timestamp,
				error, metrics, context
			FROM content_events
			WHERE display_id = $1 AND timestamp >= $2
			ORDER BY timestamp DESC
		`, displayID, since)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var event content.Event
			var errorJSON, metricsJSON, contextJSON []byte

			err := rows.Scan(
				&event.ID,
				&event.DisplayID,
				&event.Type,
				&event.URL,
				&event.Timestamp,
				&errorJSON,
				&metricsJSON,
				&contextJSON,
			)
			if err != nil {
				return err
			}

			if len(errorJSON) > 0 && string(errorJSON) != "null" {
				event.Error = &content.EventError{}
				if err := json.Unmarshal(errorJSON, event.Error); err != nil {
					return err
				}
			}

			if len(metricsJSON) > 0 && string(metricsJSON) != "{}" {
				event.Metrics = &content.EventMetrics{}
				if err := json.Unmarshal(metricsJSON, event.Metrics); err != nil {
					return err
				}
			}

			if len(contextJSON) > 0 && string(contextJSON) != "{}" {
				if err := json.Unmarshal(contextJSON, &event.Context); err != nil {
					return err
				}
			}

			events = append(events, event)
		}

		return rows.Err()
	})

	if err != nil {
		return nil, database.MapError(err, op)
	}

	return events, nil
}
