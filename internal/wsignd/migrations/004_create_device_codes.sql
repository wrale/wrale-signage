-- Migration: 004
-- Description: Create device codes table

CREATE TABLE device_codes (
    id              UUID PRIMARY KEY,
    device_code     TEXT NOT NULL UNIQUE,
    user_code       TEXT NOT NULL UNIQUE,
    expires_at      TIMESTAMP WITH TIME ZONE NOT NULL,
    poll_interval   INTEGER NOT NULL,
    activated       BOOLEAN NOT NULL DEFAULT false,
    activated_at    TIMESTAMP WITH TIME ZONE,
    display_id      UUID REFERENCES displays(id),
    created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes for lookups
CREATE INDEX device_codes_device_code_idx ON device_codes (device_code);
CREATE INDEX device_codes_user_code_idx ON device_codes (user_code);
CREATE INDEX device_codes_expires_at_idx ON device_codes (expires_at);