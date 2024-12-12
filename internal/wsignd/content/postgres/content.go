package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/wrale/wrale-signage/api/types/v1alpha1"
	"github.com/wrale/wrale-signage/internal/wsignd/database"
)

// ListContent implements content.Repository.ListContent
func (r *repository) ListContent(ctx context.Context) ([]v1alpha1.ContentSource, error) {
	const op = "ContentRepository.ListContent"
	var sources []v1alpha1.ContentSource

	err := database.RunInTx(ctx, r.db, &database.TxOptions{ReadOnly: true}, func(tx *database.Tx) error {
		rows, err := tx.QueryContext(ctx, `
			SELECT name, url, type, properties, last_validated, is_healthy, version, 
			       COALESCE(EXTRACT(EPOCH FROM playback_duration)::bigint * 1000000000, 0) as playback_duration_ns
			FROM content_sources
			ORDER BY name
		`)
		if err != nil {
			return fmt.Errorf("query failed: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var source v1alpha1.ContentSource
			var props []byte
			var durationNs int64 // Duration in nanoseconds

			err := rows.Scan(
				&source.ObjectMeta.Name,
				&source.Spec.URL,
				&source.Spec.Type,
				&props,
				&source.Status.LastValidated,
				&source.Status.IsHealthy,
				&source.Status.Version,
				&durationNs,
			)
			if err != nil {
				return fmt.Errorf("row scan failed: %w", err)
			}

			// Convert duration from nanoseconds
			source.Spec.PlaybackDuration = time.Duration(durationNs)

			if err := json.Unmarshal(props, &source.Spec.Properties); err != nil {
				return fmt.Errorf("properties unmarshal failed: %w", err)
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

// CreateContent implements content.Repository.CreateContent
func (r *repository) CreateContent(ctx context.Context, content *v1alpha1.ContentSource) error {
	const op = "ContentRepository.CreateContent"

	err := database.RunInTx(ctx, r.db, nil, func(tx *database.Tx) error {
		props, err := json.Marshal(content.Spec.Properties)
		if err != nil {
			return err
		}

		// Convert duration to interval string for PostgreSQL
		interval := fmt.Sprintf("%d seconds", int(content.Spec.PlaybackDuration.Seconds()))

		_, err = tx.ExecContext(ctx, `
			INSERT INTO content_sources (
				name, url, type, properties, 
				last_validated, is_healthy, version,
				playback_duration
			) VALUES ($1, $2, $3, $4, $5, $6, $7,
			    $8::interval)  -- Explicitly cast to interval
		`,
			content.ObjectMeta.Name,
			content.Spec.URL,
			content.Spec.Type,
			props,
			content.Status.LastValidated,
			content.Status.IsHealthy,
			content.Status.Version,
			interval,
		)
		return err
	})

	if err != nil {
		return database.MapError(err, op)
	}

	return nil
}

// UpdateContent implements content.Repository.UpdateContent
func (r *repository) UpdateContent(ctx context.Context, content *v1alpha1.ContentSource) error {
	const op = "ContentRepository.UpdateContent"

	err := database.RunInTx(ctx, r.db, nil, func(tx *database.Tx) error {
		props, err := json.Marshal(content.Spec.Properties)
		if err != nil {
			return err
		}

		// Convert duration to interval string for PostgreSQL
		interval := fmt.Sprintf("%d seconds", int(content.Spec.PlaybackDuration.Seconds()))

		result, err := tx.ExecContext(ctx, `
			UPDATE content_sources SET
				url = $2,
				type = $3,
				properties = $4,
				last_validated = $5,
				is_healthy = $6,
				version = $7,
				playback_duration = $8::interval,
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
			interval,
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

// DeleteContent implements content.Repository.DeleteContent
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

// GetContent implements content.Repository.GetContent
func (r *repository) GetContent(ctx context.Context, name string) (*v1alpha1.ContentSource, error) {
	const op = "ContentRepository.GetContent"

	var source v1alpha1.ContentSource
	source.ObjectMeta.Name = name

	err := database.RunInTx(ctx, r.db, &database.TxOptions{ReadOnly: true}, func(tx *database.Tx) error {
		var props []byte
		var durationNs int64 // Duration in nanoseconds

		err := tx.QueryRowContext(ctx, `
			SELECT url, type, properties, last_validated, is_healthy, version,
			       COALESCE(EXTRACT(EPOCH FROM playback_duration)::bigint * 1000000000, 0) as playback_duration_ns
			FROM content_sources 
			WHERE name = $1
		`, name).Scan(
			&source.Spec.URL,
			&source.Spec.Type,
			&props,
			&source.Status.LastValidated,
			&source.Status.IsHealthy,
			&source.Status.Version,
			&durationNs,
		)
		if err != nil {
			return err
		}

		// Convert duration from nanoseconds
		source.Spec.PlaybackDuration = time.Duration(durationNs)

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
