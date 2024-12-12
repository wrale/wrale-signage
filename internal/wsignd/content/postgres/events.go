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

// SaveEvent implements content.Repository.SaveEvent
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

	metricsJSON, err := json.Marshal(metrics)
	if err != nil {
		return database.MapError(err, op)
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
			return sql.ErrNoRows
		}

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

// GetDisplayEvents implements content.Repository.GetDisplayEvents
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

			if len(contextJSON) > 0 {
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
