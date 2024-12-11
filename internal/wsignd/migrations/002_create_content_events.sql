-- Migration: 002
-- Description: Create content events table
-- Version: 1

-- Up Migration
CREATE TABLE content_events (
    id          UUID PRIMARY KEY,
    display_id  UUID NOT NULL REFERENCES displays(id),
    type        TEXT NOT NULL,
    url         TEXT NOT NULL,
    timestamp   TIMESTAMP WITH TIME ZONE NOT NULL,
    error       JSONB,
    metrics     JSONB,
    context     JSONB,
    created_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes for common queries
CREATE INDEX content_events_display_id_idx ON content_events (display_id);
CREATE INDEX content_events_url_idx ON content_events (url);
CREATE INDEX content_events_timestamp_idx ON content_events (timestamp);
CREATE INDEX content_events_type_idx ON content_events (type);

-- Add partitioning by time for efficient cleanup
-- Note: Requires appropriate retention policy implementation
CREATE INDEX content_events_cleanup_idx ON content_events (created_at);

-- Down Migration
DROP TABLE IF EXISTS content_events;