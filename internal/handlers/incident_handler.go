package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sharonowusu/servicepulse/internal/repository"
)

// IncidentHandler serves incidents for a service.
type IncidentHandler struct {
	repo repository.IncidentStore
}

// NewIncidentHandler creates an IncidentHandler.
func NewIncidentHandler(repo repository.IncidentStore) *IncidentHandler {
	return &IncidentHandler{repo: repo}
}

// List handles GET /api/services/{id}/incidents.
func (h *IncidentHandler) List(w http.ResponseWriter, r *http.Request) {
	serviceID := chi.URLParam(r, "id")

	incidents, err := h.repo.GetByServiceID(r.Context(), serviceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch incidents")
		return
	}

	writeJSON(w, http.StatusOK, incidents)
}
