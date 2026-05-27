-- Migration 002: store every health check result

CREATE TABLE IF NOT EXISTS check_results (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    service_id       UUID        NOT NULL REFERENCES services(id) ON DELETE CASCADE,
    status_code      INTEGER,
    response_time_ms INTEGER     NOT NULL,
    is_success       BOOLEAN     NOT NULL,
    error_message    TEXT,
    checked_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index used heavily by the worker and the recent-checks endpoint.
CREATE INDEX IF NOT EXISTS idx_check_results_service_id ON check_results(service_id);
CREATE INDEX IF NOT EXISTS idx_check_results_checked_at  ON check_results(checked_at DESC);
