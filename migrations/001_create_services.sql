-- Migration 001: create the services table
-- Each row represents one API/website being monitored.

CREATE TABLE IF NOT EXISTS services (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                  TEXT        NOT NULL,
    url                   TEXT        NOT NULL,
    method                TEXT        NOT NULL DEFAULT 'GET',
    expected_status_code  INTEGER     NOT NULL DEFAULT 200,
    check_interval_seconds INTEGER    NOT NULL DEFAULT 60,
    is_active             BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
