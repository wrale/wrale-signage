package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wrale/wrale-signage/internal/wsignd/content"
	"github.com/wrale/wrale-signage/internal/wsignd/testutil"
)

func TestSaveEvent(t *testing.T) {
	db, cleanup := testutil.SetupTestDB(t)
	defer cleanup()

	repo := NewRepository(db)
	ctx := context.Background()

	// Create a test display first
	displayID := uuid.New()
	_, err := db.Exec(`
		INSERT INTO displays (id, name, site_id, zone, position, state, last_seen)
		VALUES ($1, 'test-display', 'site-1', 'zone-1', 'pos-1', 'ACTIVE', NOW())
	`, displayID)
	require.NoError(t, err)

	tests := []struct {
		name    string
		event   content.Event
		wantErr bool
	}{
		{
			name: "basic_event",
			event: content.Event{
				ID:        uuid.New(),
				DisplayID: displayID,
				Type:      content.EventContentLoaded,
				URL:       "https://example.com/content",
				Timestamp: time.Now(),
			},
			wantErr: false,
		},
		{
			name: "event_with_metrics",
			event: content.Event{
				ID:        uuid.New(),
				DisplayID: displayID,
				Type:      content.EventContentLoaded,
				URL:       "https://example.com/content",
				Timestamp: time.Now(),
				Metrics: &content.EventMetrics{
					LoadTime:   1000,
					RenderTime: 500,
				},
			},
			wantErr: false,
		},
		{
			name: "event_with_error",
			event: content.Event{
				ID:        uuid.New(),
				DisplayID: displayID,
				Type:      content.EventContentError,
				URL:       "https://example.com/content",
				Timestamp: time.Now(),
				Error: &content.EventError{
					Code:    "LOAD_FAILED",
					Message: "Failed to load content",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.SaveEvent(ctx, tt.event)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			// Verify event was saved
			var count int
			err = db.QueryRow("SELECT COUNT(*) FROM content_events WHERE id = $1", tt.event.ID).Scan(&count)
			assert.NoError(t, err)
			assert.Equal(t, 1, count)
		})
	}
}

func TestGetURLMetrics(t *testing.T) {
	db, cleanup := testutil.SetupTestDB(t)
	defer cleanup()

	repo := NewRepository(db)
	ctx := context.Background()
	displayID := uuid.New()
	url := "https://example.com/content"
	since := time.Now().Add(-24 * time.Hour)

	// Create test display
	_, err := db.Exec(`
		INSERT INTO displays (id, name, site_id, zone, position, state, last_seen)
		VALUES ($1, 'test-display', 'site-1', 'zone-1', 'pos-1', 'ACTIVE', NOW())
	`, displayID)
	require.NoError(t, err)

	// Insert test events
	events := []content.Event{
		{
			ID:        uuid.New(),
			DisplayID: displayID,
			Type:      content.EventContentLoaded,
			URL:       url,
			Timestamp: time.Now(),
			Metrics: &content.EventMetrics{
				LoadTime:   1000,
				RenderTime: 500,
			},
		},
		{
			ID:        uuid.New(),
			DisplayID: displayID,
			Type:      content.EventContentError,
			URL:       url,
			Timestamp: time.Now(),
			Error: &content.EventError{
				Code:    "LOAD_FAILED",
				Message: "Failed to load content",
			},
		},
	}

	for _, event := range events {
		err := repo.SaveEvent(ctx, event)
		require.NoError(t, err)
	}

	metrics, err := repo.GetURLMetrics(ctx, url, since)
	require.NoError(t, err)
	assert.NotNil(t, metrics)
	assert.Equal(t, url, metrics.URL)
	assert.Equal(t, int64(1), metrics.LoadCount)
	assert.Equal(t, int64(1), metrics.ErrorCount)
	assert.Equal(t, float64(1000), metrics.AvgLoadTime)
	assert.Equal(t, float64(500), metrics.AvgRenderTime)
	assert.Contains(t, metrics.ErrorRates, "LOAD_FAILED")
}
