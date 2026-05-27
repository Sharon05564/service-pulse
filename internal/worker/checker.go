package worker

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/sharonowusu/servicepulse/internal/models"
	"github.com/sharonowusu/servicepulse/internal/repository"
)

// HealthChecker is the background worker that periodically checks all active
// services and persists the results.
type HealthChecker struct {
	serviceRepo  repository.ServiceStore
	checkRepo    repository.CheckStore
	incidentRepo repository.IncidentStore
	httpClient   *http.Client
	logger       *slog.Logger

	// lastChecked tracks when each service was last checked so the worker
	// respects each service's individual check_interval_seconds.
	mu          sync.Mutex
	lastChecked map[string]time.Time
}

// NewHealthChecker creates a HealthChecker with its dependencies.
func NewHealthChecker(
	serviceRepo repository.ServiceStore,
	checkRepo repository.CheckStore,
	incidentRepo repository.IncidentStore,
	logger *slog.Logger,
) *HealthChecker {
	return &HealthChecker{
		serviceRepo:  serviceRepo,
		checkRepo:    checkRepo,
		incidentRepo: incidentRepo,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger:      logger,
		lastChecked: make(map[string]time.Time),
	}
}

// Run starts the checker loop. It blocks until ctx is cancelled.
// Call this in a goroutine from main.
func (hc *HealthChecker) Run(ctx context.Context) {
	hc.logger.Info("health checker started")

	// The outer tick is a short poll interval so the worker can react quickly
	// to newly-registered services without needing an explicit signal.
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			hc.logger.Info("health checker stopped")
			return
		case <-ticker.C:
			hc.runChecks(ctx)
		}
	}
}

// runChecks loads active services and fires a goroutine for each one that is
// due for a check. A WaitGroup ensures we don't start a new round before the
// current one finishes, keeping concurrent goroutine count proportional to the
// number of services.
func (hc *HealthChecker) runChecks(ctx context.Context) {
	services, err := hc.serviceRepo.GetAllActive(ctx)
	if err != nil {
		hc.logger.Error("failed to load active services", "error", err)
		return
	}

	var wg sync.WaitGroup
	for _, svc := range services {
		if !hc.isDue(svc) {
			continue
		}

		wg.Add(1)
		go func(s models.Service) {
			defer wg.Done()
			hc.checkService(ctx, s)
		}(svc)
	}
	wg.Wait()
}

// isDue returns true when the service's check_interval_seconds has elapsed
// since it was last checked.
func (hc *HealthChecker) isDue(s models.Service) bool {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	last, seen := hc.lastChecked[s.ID]
	if !seen {
		return true
	}
	return time.Since(last) >= time.Duration(s.CheckIntervalSeconds)*time.Second
}

// markChecked records the current time as the last check time for a service.
func (hc *HealthChecker) markChecked(serviceID string) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.lastChecked[serviceID] = time.Now()
}

// checkService performs a single HTTP request to the service URL, saves the
// result, and manages incident state.
func (hc *HealthChecker) checkService(ctx context.Context, s models.Service) {
	defer hc.markChecked(s.ID)

	result := hc.probe(s)

	if err := hc.checkRepo.SaveResult(ctx, result); err != nil {
		hc.logger.Error("failed to save check result", "service", s.Name, "error", err)
		return
	}

	if err := hc.handleIncident(ctx, s, result); err != nil {
		hc.logger.Error("incident handling error", "service", s.Name, "error", err)
	}

	hc.logger.Info("checked service",
		"name", s.Name,
		"success", result.IsSuccess,
		"latency_ms", result.ResponseTimeMs,
	)
}

// probe calls probeWithClient using the checker's configured HTTP client.
func (hc *HealthChecker) probe(s models.Service) *models.CheckResult {
	return probeWithClient(hc.httpClient, s)
}

// Probe sends an HTTP request to s.URL and returns a CheckResult.
// It is exported so tests can call it directly with a test HTTP server.
// It always returns a result — even on network error — so the caller has a
// record of every failure.
func Probe(s models.Service) *models.CheckResult {
	return probeWithClient(&http.Client{Timeout: 10 * time.Second}, s)
}

// probeWithClient is the shared implementation used by both Probe and the
// checker's internal probe method.
func probeWithClient(client *http.Client, s models.Service) *models.CheckResult {
	result := &models.CheckResult{
		ServiceID: s.ID,
	}

	req, err := http.NewRequest(s.Method, s.URL, nil)
	if err != nil {
		result.IsSuccess = false
		result.ErrorMessage = sql.NullString{String: fmt.Sprintf("build request: %s", err), Valid: true}
		return result
	}

	start := time.Now()
	resp, err := client.Do(req)
	result.ResponseTimeMs = int(time.Since(start).Milliseconds())

	if err != nil {
		result.IsSuccess = false
		result.ErrorMessage = sql.NullString{String: err.Error(), Valid: true}
		return result
	}
	defer resp.Body.Close()

	result.StatusCode = sql.NullInt64{Int64: int64(resp.StatusCode), Valid: true}
	result.IsSuccess = resp.StatusCode == s.ExpectedStatusCode

	if !result.IsSuccess {
		msg := fmt.Sprintf("expected status %d, got %d", s.ExpectedStatusCode, resp.StatusCode)
		result.ErrorMessage = sql.NullString{String: msg, Valid: true}
	}

	return result
}

// handleIncident opens a new incident on failure, or resolves the existing one
// on recovery.
func (hc *HealthChecker) handleIncident(ctx context.Context, s models.Service, result *models.CheckResult) error {
	if result.IsSuccess {
		return hc.resolveIfOpen(ctx, s.ID)
	}
	return hc.openIfNone(ctx, s, result)
}

// openIfNone creates an incident only when there is not already an open one.
func (hc *HealthChecker) openIfNone(ctx context.Context, s models.Service, result *models.CheckResult) error {
	hasOpen, err := hc.incidentRepo.HasOpenIncident(ctx, s.ID)
	if err != nil {
		return err
	}
	if hasOpen {
		return nil // already tracking this outage
	}

	reason := "service check failed"
	if result.ErrorMessage.Valid {
		reason = result.ErrorMessage.String
	}

	_, err = hc.incidentRepo.Create(ctx, s.ID, reason)
	if err != nil {
		return fmt.Errorf("open incident: %w", err)
	}

	hc.logger.Warn("incident opened", "service", s.Name, "reason", reason)
	return nil
}

// resolveIfOpen closes the open incident for a service that just recovered.
func (hc *HealthChecker) resolveIfOpen(ctx context.Context, serviceID string) error {
	incident, err := hc.incidentRepo.GetOpenByServiceID(ctx, serviceID)
	if err != nil {
		// sql.ErrNoRows means no open incident — that is fine.
		if err.Error() == "get open incident: sql: no rows in result set" {
			return nil
		}
		return err
	}

	if err := hc.incidentRepo.Resolve(ctx, incident.ID); err != nil {
		return fmt.Errorf("resolve incident: %w", err)
	}

	hc.logger.Info("incident resolved", "service_id", serviceID)
	return nil
}
