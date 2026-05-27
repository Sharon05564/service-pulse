package handlers

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// RouterDeps groups all handler dependencies so NewRouter stays readable.
type RouterDeps struct {
	Service  *ServiceHandler
	Check    *CheckHandler
	Incident *IncidentHandler
	Metrics  *MetricsHandler
}

// NewRouter builds and returns the application's HTTP router.
func NewRouter(deps RouterDeps) http.Handler {
	r := chi.NewRouter()

	// Standard middleware stack.
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	// System health check — no auth required.
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Route("/api/services", func(r chi.Router) {
		r.Post("/", deps.Service.Create)
		r.Get("/", deps.Service.List)

		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", deps.Service.Get)
			r.Put("/", deps.Service.Update)
			r.Delete("/", deps.Service.Delete)

			r.Get("/checks", deps.Check.List)
			r.Get("/incidents", deps.Incident.List)
			r.Get("/summary", deps.Metrics.Summary)
		})
	})

	return r
}
