package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/sharonowusu/servicepulse/internal/models"
)

// ServiceRepository handles all database operations for the services table.
type ServiceRepository struct {
	db *sqlx.DB
}

// NewServiceRepository creates a ServiceRepository backed by db.
func NewServiceRepository(db *sqlx.DB) *ServiceRepository {
	return &ServiceRepository{db: db}
}

// Create inserts a new service and returns the full row as saved in the DB.
func (r *ServiceRepository) Create(ctx context.Context, req models.CreateServiceRequest) (*models.Service, error) {
	// Apply defaults for optional fields.
	method := req.Method
	if method == "" {
		method = "GET"
	}

	expectedStatus := req.ExpectedStatusCode
	if expectedStatus == 0 {
		expectedStatus = 200
	}

	interval := req.CheckIntervalSeconds
	if interval == 0 {
		interval = 60
	}

	const q = `
		INSERT INTO services (name, url, method, expected_status_code, check_interval_seconds)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING *`

	var s models.Service
	err := r.db.GetContext(ctx, &s, q, req.Name, req.URL, method, expectedStatus, interval)
	if err != nil {
		return nil, fmt.Errorf("create service: %w", err)
	}
	return &s, nil
}

// GetAll returns every service ordered by creation date (newest first).
func (r *ServiceRepository) GetAll(ctx context.Context) ([]models.Service, error) {
	const q = `SELECT * FROM services ORDER BY created_at DESC`
	var services []models.Service
	if err := r.db.SelectContext(ctx, &services, q); err != nil {
		return nil, fmt.Errorf("get all services: %w", err)
	}
	return services, nil
}

// GetByID returns a single service, or an error if it does not exist.
func (r *ServiceRepository) GetByID(ctx context.Context, id string) (*models.Service, error) {
	const q = `SELECT * FROM services WHERE id = $1`
	var s models.Service
	if err := r.db.GetContext(ctx, &s, q, id); err != nil {
		return nil, fmt.Errorf("get service by id: %w", err)
	}
	return &s, nil
}

// Update replaces editable fields and bumps updated_at.
func (r *ServiceRepository) Update(ctx context.Context, id string, req models.UpdateServiceRequest) (*models.Service, error) {
	const q = `
		UPDATE services
		SET name                  = $1,
		    url                   = $2,
		    method                = $3,
		    expected_status_code  = $4,
		    check_interval_seconds = $5,
		    is_active             = $6,
		    updated_at            = $7
		WHERE id = $8
		RETURNING *`

	var s models.Service
	err := r.db.GetContext(ctx, &s, q,
		req.Name, req.URL, req.Method, req.ExpectedStatusCode,
		req.CheckIntervalSeconds, *req.IsActive, time.Now(), id,
	)
	if err != nil {
		return nil, fmt.Errorf("update service: %w", err)
	}
	return &s, nil
}

// Delete removes a service by ID (cascades to check_results and incidents).
func (r *ServiceRepository) Delete(ctx context.Context, id string) error {
	const q = `DELETE FROM services WHERE id = $1`
	result, err := r.db.ExecContext(ctx, q, id)
	if err != nil {
		return fmt.Errorf("delete service: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("service not found")
	}
	return nil
}

// GetAllActive returns only services where is_active = true.
// Used by the background worker to know what to check.
func (r *ServiceRepository) GetAllActive(ctx context.Context) ([]models.Service, error) {
	const q = `SELECT * FROM services WHERE is_active = TRUE`
	var services []models.Service
	if err := r.db.SelectContext(ctx, &services, q); err != nil {
		return nil, fmt.Errorf("get active services: %w", err)
	}
	return services, nil
}
