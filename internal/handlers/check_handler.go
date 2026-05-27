package handlers

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/sharonowusu/servicepulse/internal/repository"
)

// CheckHandler serves recent health check results for a service.
type CheckHandler struct {
	repo repository.CheckStore
}

// NewCheckHandler creates a CheckHandler.
func NewCheckHandler(repo repository.CheckStore) *CheckHandler {
	return &CheckHandler{repo: repo}
}

// List handles GET /api/services/{id}/checks.
// Accepts an optional ?limit= query parameter (default 50, max 200).
func (h *CheckHandler) List(w http.ResponseWriter, r *http.Request) {
	serviceID := chi.URLParam(r, "id")

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}
	if limit > 200 {
		limit = 200
	}

	results, err := h.repo.GetByServiceID(r.Context(), serviceID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch check results")
		return
	}

	writeJSON(w, http.StatusOK, results)
}
