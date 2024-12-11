-- Create content_sources table
CREATE TABLE content_sources (
    name TEXT PRIMARY KEY,
    url TEXT NOT NULL,
    type TEXT NOT NULL,
    properties JSONB NOT NULL DEFAULT '{}',
    last_validated TIMESTAMP WITH TIME ZONE NOT NULL,
    is_healthy BOOLEAN NOT NULL DEFAULT false,
    version INTEGER NOT NULL DEFAULT 1,
    playback_duration INTERVAL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Add indexes for content lookups
CREATE INDEX content_sources_type_idx ON content_sources(type);
CREATE INDEX content_sources_url_idx ON content_sources(url);
CREATE INDEX content_sources_healthy_idx ON content_sources(is_healthy);

-- Trigger to update timestamps
CREATE TRIGGER content_sources_update_timestamps
    BEFORE UPDATE ON content_sources
    FOR EACH ROW
    EXECUTE FUNCTION update_timestamp();
