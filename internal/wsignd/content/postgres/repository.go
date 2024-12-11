package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/wrale/wrale-signage/api/types/v1alpha1"
	"github.com/wrale/wrale-signage/internal/wsignd/content"
	"github.com/wrale/wrale-signage/internal/wsignd/database"
)

type repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) content.Repository {
	return &repository{db: db}
}

func (r *repository) CreateContent(ctx context.Context, content *v1alpha1.ContentSource) error {
	const op = "ContentRepository.CreateContent"

	err := database.RunInTx(ctx, r.db, nil, func(tx *database.Tx) error {
		props, err := json.Marshal(content.Spec.Properties)
		if err != nil {
			return err
		}

		_, err = tx.ExecContext(ctx, `
			INSERT INTO content_sources (
				name, url, type, properties, 
				last_validated, is_healthy, version,
				playback_duration
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`,
			content.ObjectMeta.Name,
			content.Spec.URL,
			content.Spec.Type,
			props,
			content.Status.LastValidated,
			content.Status.IsHealthy,
			content.Status.Version,
			content.Spec.PlaybackDuration,
		)
		return err
	})

	if err != nil {
		return database.MapError(err, op)
	}

	return nil
}

func (r *repository) UpdateContent(ctx context.Context, content *v1alpha1.ContentSource) error {
	const op = "ContentRepository.UpdateContent"

	err := database.RunInTx(ctx, r.db, nil, func(tx *database.Tx) error {
		props, err := json.Marshal(content.Spec.Properties)
		if err != nil {
			return err
		}

		result, err := tx.ExecContext(ctx, `
			UPDATE content_sources SET
				url = $2,
				type = $3,
				properties = $4,
				last_validated = $5,
				is_healthy = $6,
				version = $7,
				playback_duration = $8,
				updated_at = NOW()
			WHERE name = $1
		`,
			content.ObjectMeta.Name,
			content.Spec.URL,
			content.Spec.Type,
			props,
			content.Status.LastValidated,
			content.Status.IsHealthy,
			content.Status.Version,
			content.Spec.PlaybackDuration,
		)
		if err != nil {
			return err
		}

		rows, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if rows == 0 {
			return sql.ErrNoRows
		}

		return nil
	})

	if err != nil {
		return database.MapError(err, op)
	}

	return nil
}

func (r *repository) DeleteContent(ctx context.Context, name string) error {
	const op = "ContentRepository.DeleteContent"

	err := database.RunInTx(ctx, r.db, nil, func(tx *database.Tx) error {
		result, err := tx.ExecContext(ctx, `DELETE FROM content_sources WHERE name = $1`, name)
		if err != nil {
			return err
		}

		rows, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if rows == 0 {
			return sql.ErrNoRows
		}

		return nil
	})

	if err != nil {
		return database.MapError(err, op)
	}

	return nil
}

func (r *repository) GetContent(ctx context.Context, name string) (*v1alpha1.ContentSource, error) {
	const op = "ContentRepository.GetContent"

	var source v1alpha1.ContentSource
	source.ObjectMeta.Name = name

	err := database.RunInTx(ctx, r.db, &database.TxOptions{ReadOnly: true}, func(tx *database.Tx) error {
		var props []byte
		err := tx.QueryRowContext(ctx, `
			SELECT url, type, properties, last_validated, is_healthy, version, playback_duration
			FROM content_sources 
			WHERE name = $1
		`, name).Scan(
			&source.Spec.URL,
			&source.Spec.Type,
			&props,
			&source.Status.LastValidated,
			&source.Status.IsHealthy,
			&source.Status.Version,
			&source.Spec.PlaybackDuration,
		)
		if err != nil {
			return err
		}

		return json.Unmarshal(props, &source.Spec.Properties)
	})

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, database.MapError(err, op)
		}
		return nil, database.MapError(err, op)
	}

	return &source, nil
}

func (r *repository) ListContent(ctx context.Context) ([]v1alpha1.ContentSource, error) {
	const op = "ContentRepository.ListContent"

	var sources []v1alpha1.ContentSource

	err := database.RunInTx(ctx, r.db, &database.TxOptions{ReadOnly: true}, func(tx *database.Tx) error {
		rows, err := tx.QueryContext(ctx, `
			SELECT name, url, type, properties, last_validated, is_healthy, version, playback_duration
			FROM content_sources
			ORDER BY name
		`)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var source v1alpha1.ContentSource
			var props []byte

			err := rows.Scan(
				&source.ObjectMeta.Name,
				&source.Spec.URL,
				&source.Spec.Type,
				&props,
				&source.Status.LastValidated,
				&source.Status.IsHealthy,
				&source.Status.Version,
				&source.Spec.PlaybackDuration,
			)
			if err != nil {
				return err
			}

			if err := json.Unmarshal(props, &source.Spec.Properties); err != nil {
				return err
			}

			sources = append(sources, source)
		}

		return rows.Err()
	})

	if err != nil {
		return nil, database.MapError(err, op)
	}

	return sources, nil
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
			SELECT 
				COALESCE(AVG((metrics->>'loadTime')::numeric), 0),
				COALESCE(AVG((metrics->>'renderTime')::numeric), 0)
			FROM content_events 
			WHERE url = $1 
				AND timestamp >= $2
				AND type = 'CONTENT_LOADED'
				AND metrics IS NOT NULL
				AND metrics->>'loadTime' IS NOT NULL
				AND metrics->>'renderTime' IS NOT NULL
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
					AND error->>'code' IS NOT NULL
				GROUP BY error->>'code'
			),
			total AS (
				SELECT COUNT(*)::float8 as total_count
				FROM content_events 
				WHERE url = $1 AND timestamp >= $2
			)
			SELECT 
				error_code,
				COALESCE(code_count / NULLIF(total_count, 0), 0)
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
