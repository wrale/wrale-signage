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

	metrics, err := json.Marshal(event.Metrics)
	if err != nil {
		return database.MapError(err, op)
	}

	var errorJSON []byte
	if event.Error != nil {
		errorJSON, err = json.Marshal(event.Error)
		if err != nil {
			return database.MapError(err, op)
		}
	}

	context, err := json.Marshal(event.Context)
	if err != nil {
		return database.MapError(err, op)
	}

	err = database.RunInTx(ctx, r.db, nil, func(tx *database.Tx) error {
		_, err := tx.ExecContext(ctx, `
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
			metrics,
			context,
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
		// Get last seen timestamp
		err := tx.QueryRowContext(ctx, `
			SELECT EXTRACT(EPOCH FROM MAX(timestamp))
			FROM content_events 
			WHERE url = $1
		`, url).Scan(&metrics.LastSeen)
		if err != nil && err != sql.ErrNoRows {
			return err
		}

		// Get load and error counts
		err = tx.QueryRowContext(ctx, `
			SELECT 
				COUNT(*) FILTER (WHERE type = 'CONTENT_LOADED'),
				COUNT(*) FILTER (WHERE type = 'CONTENT_ERROR')
			FROM content_events 
			WHERE url = $1 AND timestamp >= $2
		`, url, since).Scan(&metrics.LoadCount, &metrics.ErrorCount)
		if err != nil && err != sql.ErrNoRows {
			return err
		}

		// Get average timing metrics
		err = tx.QueryRowContext(ctx, `
			SELECT 
				AVG((metrics->>'loadTime')::bigint),
				AVG((metrics->>'renderTime')::bigint)
			FROM content_events 
			WHERE url = $1 
				AND timestamp >= $2
				AND metrics IS NOT NULL
		`, url, since).Scan(&metrics.AvgLoadTime, &metrics.AvgRenderTime)
		if err != nil && err != sql.ErrNoRows {
			return err
		}

		// Get error rates by code
		rows, err := tx.QueryContext(ctx, `
			SELECT 
				error->>'code' as error_code,
				COUNT(*)::float / (
					SELECT COUNT(*)::float 
					FROM content_events 
					WHERE url = $1 AND timestamp >= $2
				) as error_rate
			FROM content_events 
			WHERE url = $1 
				AND timestamp >= $2
				AND type = 'CONTENT_ERROR'
				AND error IS NOT NULL
			GROUP BY error->>'code'
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

			if len(errorJSON) > 0 {
				event.Error = &content.EventError{}
				if err := json.Unmarshal(errorJSON, event.Error); err != nil {
					return err
				}
			}

			if len(metricsJSON) > 0 {
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