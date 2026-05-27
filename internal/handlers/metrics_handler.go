package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sharonowusu/servicepulse/internal/models"
	"github.com/sharonowusu/servicepulse/internal/repository"
)

// MetricsHandler serves the summary endpoint for a service.
type MetricsHandler struct {
	checkRepo    repository.CheckStore
	incidentRepo repository.IncidentStore
}

// NewMetricsHandler creates a MetricsHandler.
func NewMetricsHandler(checkRepo repository.CheckStore, incidentRepo repository.IncidentStore) *MetricsHandler {
	return &MetricsHandler{checkRepo: checkRepo, incidentRepo: incidentRepo}
}

// Summary handles GET /api/services/{id}/summary.
func (h *MetricsHandler) Summary(w http.ResponseWriter, r *http.Request) {
	serviceID := chi.URLParam(r, "id")
	ctx := r.Context()

	total, successful, avgLatency, lastChecked, err := h.checkRepo.GetSummaryStats(ctx, serviceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch summary stats")
		return
	}

	hasIncident, err := h.incidentRepo.HasOpenIncident(ctx, serviceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to check incidents")
		return
	}

	failed := total - successful
	var uptime float64
	if total > 0 {
		uptime = float64(successful) / float64(total) * 100
	}

	summary := models.ServiceSummary{
		ServiceID:        serviceID,
		TotalChecks:      total,
		SuccessfulChecks: successful,
		FailedChecks:     failed,
		UptimePercent:    uptime,
		AvgLatencyMs:     avgLatency,
		ActiveIncident:   hasIncident,
		LastCheckedAt:    lastChecked,
	}

	writeJSON(w, http.StatusOK, summary)
}
