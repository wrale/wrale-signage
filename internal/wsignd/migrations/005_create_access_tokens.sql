CREATE TABLE access_tokens (
    id UUID PRIMARY KEY,
    display_id UUID NOT NULL REFERENCES displays(id) ON DELETE CASCADE,
    access_token_hash BYTEA NOT NULL,
    refresh_token_hash BYTEA NOT NULL,
    access_token_expires_at TIMESTAMPTZ NOT NULL,
    refresh_token_expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX access_tokens_display_id_idx ON access_tokens (display_id);
CREATE INDEX access_tokens_access_token_hash_idx ON access_tokens (access_token_hash);
CREATE INDEX access_tokens_refresh_token_hash_idx ON access_tokens (refresh_token_hash);
CREATE INDEX access_tokens_expires_at_idx ON access_tokens (access_token_expires_at);

-- Add constraint to ensure only one active token set per display
CREATE UNIQUE INDEX access_tokens_display_id_active_idx 
ON access_tokens (display_id) 
WHERE access_token_expires_at > NOW();

COMMENT ON TABLE access_tokens IS 'Stores OAuth tokens for authenticated displays';
COMMENT ON COLUMN access_tokens.access_token_hash IS 'SHA-256 hash of access token';
COMMENT ON COLUMN access_tokens.refresh_token_hash IS 'SHA-256 hash of refresh token';
