-- Migration 003: track outages (incidents) per service

CREATE TABLE IF NOT EXISTS incidents (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    service_id  UUID        NOT NULL REFERENCES services(id) ON DELETE CASCADE,
    started_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ,
    reason      TEXT,
    is_resolved BOOLEAN     NOT NULL DEFAULT FALSE
);

CREATE INDEX IF NOT EXISTS idx_incidents_service_id  ON incidents(service_id);
CREATE INDEX IF NOT EXISTS idx_incidents_is_resolved ON incidents(is_resolved);
