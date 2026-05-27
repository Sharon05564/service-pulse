package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/sharonowusu/servicepulse/internal/models"
	"github.com/sharonowusu/servicepulse/internal/repository"
)

// ServiceHandler holds the dependencies for the /api/services endpoints.
type ServiceHandler struct {
	repo repository.ServiceStore
}

// NewServiceHandler creates a ServiceHandler.
func NewServiceHandler(repo repository.ServiceStore) *ServiceHandler {
	return &ServiceHandler{repo: repo}
}

// Create handles POST /api/services.
func (h *ServiceHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.CreateServiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.URL) == "" {
		writeError(w, http.StatusBadRequest, "name and url are required")
		return
	}

	service, err := h.repo.Create(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create service")
		return
	}

	writeJSON(w, http.StatusCreated, service)
}

// List handles GET /api/services.
func (h *ServiceHandler) List(w http.ResponseWriter, r *http.Request) {
	services, err := h.repo.GetAll(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch services")
		return
	}

	writeJSON(w, http.StatusOK, services)
}

// Get handles GET /api/services/{id}.
func (h *ServiceHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	service, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "service not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch service")
		return
	}

	writeJSON(w, http.StatusOK, service)
}

// Update handles PUT /api/services/{id}.
func (h *ServiceHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Load existing service to carry forward any fields not sent in the request.
	existing, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "service not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch service")
		return
	}

	var req models.UpdateServiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Fill in any unset fields with current values so a partial update is possible.
	if req.Name == "" {
		req.Name = existing.Name
	}
	if req.URL == "" {
		req.URL = existing.URL
	}
	if req.Method == "" {
		req.Method = existing.Method
	}
	if req.ExpectedStatusCode == 0 {
		req.ExpectedStatusCode = existing.ExpectedStatusCode
	}
	if req.CheckIntervalSeconds == 0 {
		req.CheckIntervalSeconds = existing.CheckIntervalSeconds
	}
	if req.IsActive == nil {
		req.IsActive = &existing.IsActive
	}

	updated, err := h.repo.Update(r.Context(), id, req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update service")
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

// Delete handles DELETE /api/services/{id}.
func (h *ServiceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.repo.Delete(r.Context(), id); err != nil {
		if err.Error() == "service not found" {
			writeError(w, http.StatusNotFound, "service not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to delete service")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
