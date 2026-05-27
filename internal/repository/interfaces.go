package repository

import (
	"context"
	"time"

	"github.com/sharonowusu/servicepulse/internal/models"
)

// ServiceStore is the interface the service handler depends on.
// The concrete *ServiceRepository satisfies it; tests can use a fake.
type ServiceStore interface {
	Create(ctx context.Context, req models.CreateServiceRequest) (*models.Service, error)
	GetAll(ctx context.Context) ([]models.Service, error)
	GetByID(ctx context.Context, id string) (*models.Service, error)
	Update(ctx context.Context, id string, req models.UpdateServiceRequest) (*models.Service, error)
	Delete(ctx context.Context, id string) error
	GetAllActive(ctx context.Context) ([]models.Service, error)
}

// CheckStore is the interface the check handler and metrics handler depend on.
type CheckStore interface {
	SaveResult(ctx context.Context, result *models.CheckResult) error
	GetByServiceID(ctx context.Context, serviceID string, limit int) ([]models.CheckResult, error)
	GetSummaryStats(ctx context.Context, serviceID string) (total, successful int, avgLatency float64, lastChecked *time.Time, err error)
}

// IncidentStore is the interface the incident handler and worker depend on.
type IncidentStore interface {
	Create(ctx context.Context, serviceID, reason string) (*models.Incident, error)
	GetOpenByServiceID(ctx context.Context, serviceID string) (*models.Incident, error)
	Resolve(ctx context.Context, incidentID string) error
	GetByServiceID(ctx context.Context, serviceID string) ([]models.Incident, error)
	HasOpenIncident(ctx context.Context, serviceID string) (bool, error)
}
