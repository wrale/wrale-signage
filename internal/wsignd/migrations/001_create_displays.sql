-- Migration: 001
-- Description: Create displays table

CREATE TABLE displays (
    id              UUID PRIMARY KEY,
    name            TEXT NOT NULL UNIQUE,
    site_id         TEXT NOT NULL,
    zone            TEXT NOT NULL,
    position        TEXT NOT NULL,
    state           TEXT NOT NULL,
    last_seen       TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    version         INTEGER NOT NULL DEFAULT 1,
    properties      JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes for common queries
CREATE INDEX displays_site_zone_idx ON displays (site_id, zone);
CREATE INDEX displays_state_idx ON displays (state);
CREATE INDEX displays_last_seen_idx ON displays (last_seen);

-- Add trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_displays_updated_at
    BEFORE UPDATE ON displays
    FOR EACH ROW
    EXECUTE PROCEDURE update_updated_at_column();