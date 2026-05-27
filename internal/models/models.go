package models

import (
	"database/sql"
	"time"
)

// Service represents a monitored API or website.
type Service struct {
	ID                   string    `db:"id"                     json:"id"`
	Name                 string    `db:"name"                   json:"name"`
	URL                  string    `db:"url"                    json:"url"`
	Method               string    `db:"method"                 json:"method"`
	ExpectedStatusCode   int       `db:"expected_status_code"   json:"expected_status_code"`
	CheckIntervalSeconds int       `db:"check_interval_seconds" json:"check_interval_seconds"`
	IsActive             bool      `db:"is_active"              json:"is_active"`
	CreatedAt            time.Time `db:"created_at"             json:"created_at"`
	UpdatedAt            time.Time `db:"updated_at"             json:"updated_at"`
}

// CheckResult stores the outcome of a single health check against a service.
type CheckResult struct {
	ID             string         `db:"id"               json:"id"`
	ServiceID      string         `db:"service_id"       json:"service_id"`
	StatusCode     sql.NullInt64  `db:"status_code"      json:"status_code"`
	ResponseTimeMs int            `db:"response_time_ms" json:"response_time_ms"`
	IsSuccess      bool           `db:"is_success"       json:"is_success"`
	ErrorMessage   sql.NullString `db:"error_message"    json:"error_message,omitempty"`
	CheckedAt      time.Time      `db:"checked_at"       json:"checked_at"`
}

// Incident tracks a continuous period of failure for a service.
type Incident struct {
	ID         string         `db:"id"          json:"id"`
	ServiceID  string         `db:"service_id"  json:"service_id"`
	StartedAt  time.Time      `db:"started_at"  json:"started_at"`
	ResolvedAt sql.NullTime   `db:"resolved_at" json:"resolved_at,omitempty"`
	Reason     sql.NullString `db:"reason"      json:"reason,omitempty"`
	IsResolved bool           `db:"is_resolved" json:"is_resolved"`
}

// --- Request / Response DTOs ---

// CreateServiceRequest is the payload accepted by POST /api/services.
type CreateServiceRequest struct {
	Name                 string `json:"name"`
	URL                  string `json:"url"`
	Method               string `json:"method"`
	ExpectedStatusCode   int    `json:"expected_status_code"`
	CheckIntervalSeconds int    `json:"check_interval_seconds"`
}

// UpdateServiceRequest is the payload accepted by PUT /api/services/{id}.
type UpdateServiceRequest struct {
	Name                 string `json:"name"`
	URL                  string `json:"url"`
	Method               string `json:"method"`
	ExpectedStatusCode   int    `json:"expected_status_code"`
	CheckIntervalSeconds int    `json:"check_interval_seconds"`
	IsActive             *bool  `json:"is_active"`
}

// ServiceSummary is the response for GET /api/services/{id}/summary.
type ServiceSummary struct {
	ServiceID       string     `json:"service_id"`
	TotalChecks     int        `json:"total_checks"`
	SuccessfulChecks int       `json:"successful_checks"`
	FailedChecks    int        `json:"failed_checks"`
	UptimePercent   float64    `json:"uptime_percent"`
	AvgLatencyMs    float64    `json:"avg_latency_ms"`
	ActiveIncident  bool       `json:"active_incident"`
	LastCheckedAt   *time.Time `json:"last_checked_at"`
}
