package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/sharonowusu/servicepulse/internal/models"
)

// CheckRepository handles database operations for the check_results table.
type CheckRepository struct {
	db *sqlx.DB
}

// NewCheckRepository creates a CheckRepository backed by db.
func NewCheckRepository(db *sqlx.DB) *CheckRepository {
	return &CheckRepository{db: db}
}

// SaveResult inserts a completed check result.
func (r *CheckRepository) SaveResult(ctx context.Context, result *models.CheckResult) error {
	const q = `
		INSERT INTO check_results (service_id, status_code, response_time_ms, is_success, error_message)
		VALUES ($1, $2, $3, $4, $5)`

	_, err := r.db.ExecContext(ctx, q,
		result.ServiceID,
		result.StatusCode,
		result.ResponseTimeMs,
		result.IsSuccess,
		result.ErrorMessage,
	)
	if err != nil {
		return fmt.Errorf("save check result: %w", err)
	}
	return nil
}

// GetByServiceID returns up to limit recent results for a service.
func (r *CheckRepository) GetByServiceID(ctx context.Context, serviceID string, limit int) ([]models.CheckResult, error) {
	const q = `
		SELECT * FROM check_results
		WHERE service_id = $1
		ORDER BY checked_at DESC
		LIMIT $2`

	var results []models.CheckResult
	if err := r.db.SelectContext(ctx, &results, q, serviceID, limit); err != nil {
		return nil, fmt.Errorf("get check results: %w", err)
	}
	return results, nil
}

// GetSummaryStats returns aggregated data used by the metrics endpoint.
func (r *CheckRepository) GetSummaryStats(ctx context.Context, serviceID string) (total, successful int, avgLatency float64, lastChecked *time.Time, err error) {
	const q = `
		SELECT
		    COUNT(*)                                   AS total,
		    COALESCE(SUM(CASE WHEN is_success THEN 1 ELSE 0 END), 0) AS successful,
		    COALESCE(AVG(response_time_ms), 0)         AS avg_latency,
		    MAX(checked_at)                            AS last_checked
		FROM check_results
		WHERE service_id = $1`

	var row struct {
		Total       int        `db:"total"`
		Successful  int        `db:"successful"`
		AvgLatency  float64    `db:"avg_latency"`
		LastChecked *time.Time `db:"last_checked"`
	}

	if err = r.db.GetContext(ctx, &row, q, serviceID); err != nil {
		return 0, 0, 0, nil, fmt.Errorf("get summary stats: %w", err)
	}

	return row.Total, row.Successful, row.AvgLatency, row.LastChecked, nil
}
