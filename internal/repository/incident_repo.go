package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/sharonowusu/servicepulse/internal/models"
)

// IncidentRepository handles database operations for the incidents table.
type IncidentRepository struct {
	db *sqlx.DB
}

// NewIncidentRepository creates an IncidentRepository backed by db.
func NewIncidentRepository(db *sqlx.DB) *IncidentRepository {
	return &IncidentRepository{db: db}
}

// Create opens a new incident for a service.
func (r *IncidentRepository) Create(ctx context.Context, serviceID, reason string) (*models.Incident, error) {
	const q = `
		INSERT INTO incidents (service_id, reason)
		VALUES ($1, $2)
		RETURNING *`

	var i models.Incident
	if err := r.db.GetContext(ctx, &i, q, serviceID, reason); err != nil {
		return nil, fmt.Errorf("create incident: %w", err)
	}
	return &i, nil
}

// GetOpenByServiceID returns the unresolved incident for a service, if any.
// A service should have at most one open incident at a time.
func (r *IncidentRepository) GetOpenByServiceID(ctx context.Context, serviceID string) (*models.Incident, error) {
	const q = `
		SELECT * FROM incidents
		WHERE service_id = $1 AND is_resolved = FALSE
		ORDER BY started_at DESC
		LIMIT 1`

	var i models.Incident
	if err := r.db.GetContext(ctx, &i, q, serviceID); err != nil {
		return nil, fmt.Errorf("get open incident: %w", err)
	}
	return &i, nil
}

// Resolve marks an incident as resolved with the current timestamp.
func (r *IncidentRepository) Resolve(ctx context.Context, incidentID string) error {
	const q = `
		UPDATE incidents
		SET is_resolved = TRUE,
		    resolved_at = $1
		WHERE id = $2`

	now := time.Now()
	if _, err := r.db.ExecContext(ctx, q, now, incidentID); err != nil {
		return fmt.Errorf("resolve incident: %w", err)
	}
	return nil
}

// GetByServiceID returns all incidents for a service, newest first.
func (r *IncidentRepository) GetByServiceID(ctx context.Context, serviceID string) ([]models.Incident, error) {
	const q = `
		SELECT * FROM incidents
		WHERE service_id = $1
		ORDER BY started_at DESC`

	var incidents []models.Incident
	if err := r.db.SelectContext(ctx, &incidents, q, serviceID); err != nil {
		return nil, fmt.Errorf("get incidents: %w", err)
	}
	return incidents, nil
}

// HasOpenIncident returns true if the service currently has an unresolved incident.
func (r *IncidentRepository) HasOpenIncident(ctx context.Context, serviceID string) (bool, error) {
	const q = `SELECT COUNT(*) FROM incidents WHERE service_id = $1 AND is_resolved = FALSE`
	var count int
	if err := r.db.GetContext(ctx, &count, q, serviceID); err != nil {
		return false, fmt.Errorf("check open incident: %w", err)
	}
	return count > 0, nil
}
