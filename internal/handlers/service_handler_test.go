package handlers_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/sharonowusu/servicepulse/internal/handlers"
	"github.com/sharonowusu/servicepulse/internal/models"
)

// fakeServiceRepo satisfies repository.ServiceStore using an in-memory map.
// This lets tests run without a database.
type fakeServiceRepo struct {
	services map[string]*models.Service
	nextErr  error
}

func newFakeServiceRepo() *fakeServiceRepo {
	return &fakeServiceRepo{services: make(map[string]*models.Service)}
}

func (f *fakeServiceRepo) Create(_ context.Context, req models.CreateServiceRequest) (*models.Service, error) {
	if f.nextErr != nil {
		return nil, f.nextErr
	}
	method := req.Method
	if method == "" {
		method = "GET"
	}
	expected := req.ExpectedStatusCode
	if expected == 0 {
		expected = 200
	}
	interval := req.CheckIntervalSeconds
	if interval == 0 {
		interval = 60
	}
	s := &models.Service{
		ID:                   "test-id-1",
		Name:                 req.Name,
		URL:                  req.URL,
		Method:               method,
		ExpectedStatusCode:   expected,
		CheckIntervalSeconds: interval,
		IsActive:             true,
	}
	f.services[s.ID] = s
	return s, nil
}

func (f *fakeServiceRepo) GetAll(_ context.Context) ([]models.Service, error) {
	if f.nextErr != nil {
		return nil, f.nextErr
	}
	out := make([]models.Service, 0, len(f.services))
	for _, s := range f.services {
		out = append(out, *s)
	}
	return out, nil
}

func (f *fakeServiceRepo) GetByID(_ context.Context, id string) (*models.Service, error) {
	if f.nextErr != nil {
		return nil, f.nextErr
	}
	s, ok := f.services[id]
	if !ok {
		return nil, sql.ErrNoRows
	}
	return s, nil
}

func (f *fakeServiceRepo) Update(_ context.Context, id string, req models.UpdateServiceRequest) (*models.Service, error) {
	if f.nextErr != nil {
		return nil, f.nextErr
	}
	s, ok := f.services[id]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	s.Name = req.Name
	return s, nil
}

func (f *fakeServiceRepo) Delete(_ context.Context, id string) error {
	if f.nextErr != nil {
		return f.nextErr
	}
	if _, ok := f.services[id]; !ok {
		return fmt.Errorf("service not found")
	}
	delete(f.services, id)
	return nil
}

func (f *fakeServiceRepo) GetAllActive(_ context.Context) ([]models.Service, error) {
	return f.GetAll(context.Background())
}

// withChiParam attaches a chi URL parameter to the request context so handlers
// can call chi.URLParam without a real router.
func withChiParam(r *http.Request, key, value string) *http.Request {
	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, chiCtx))
}

// --- tests ---

func TestCreateService_Success(t *testing.T) {
	repo := newFakeServiceRepo()
	h := handlers.NewServiceHandler(repo)

	body := `{"name":"My API","url":"https://example.com","method":"GET","expected_status_code":200,"check_interval_seconds":30}`
	req := httptest.NewRequest(http.MethodPost, "/api/services", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d — body: %s", w.Code, w.Body.String())
	}

	var svc models.Service
	if err := json.NewDecoder(w.Body).Decode(&svc); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if svc.Name != "My API" {
		t.Errorf("expected name 'My API', got %q", svc.Name)
	}
	if svc.URL != "https://example.com" {
		t.Errorf("expected url 'https://example.com', got %q", svc.URL)
	}
}

func TestCreateService_MissingFields(t *testing.T) {
	repo := newFakeServiceRepo()
	h := handlers.NewServiceHandler(repo)

	body := `{"name":"","url":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/services", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCreateService_InvalidJSON(t *testing.T) {
	repo := newFakeServiceRepo()
	h := handlers.NewServiceHandler(repo)

	req := httptest.NewRequest(http.MethodPost, "/api/services", bytes.NewBufferString("{bad json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestListServices_Empty(t *testing.T) {
	repo := newFakeServiceRepo()
	h := handlers.NewServiceHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/services", nil)
	w := httptest.NewRecorder()

	h.List(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var services []models.Service
	if err := json.NewDecoder(w.Body).Decode(&services); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(services) != 0 {
		t.Errorf("expected empty list, got %d items", len(services))
	}
}

func TestListServices_AfterCreate(t *testing.T) {
	repo := newFakeServiceRepo()
	h := handlers.NewServiceHandler(repo)

	// Seed one service directly into the fake repo.
	repo.services["abc"] = &models.Service{ID: "abc", Name: "Test", URL: "https://test.com", IsActive: true}

	req := httptest.NewRequest(http.MethodGet, "/api/services", nil)
	w := httptest.NewRecorder()
	h.List(w, req)

	var services []models.Service
	_ = json.NewDecoder(w.Body).Decode(&services)
	if len(services) != 1 {
		t.Errorf("expected 1 service, got %d", len(services))
	}
}

func TestGetService_NotFound(t *testing.T) {
	repo := newFakeServiceRepo()
	h := handlers.NewServiceHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/services/missing", nil)
	req = withChiParam(req, "id", "missing")
	w := httptest.NewRecorder()

	h.Get(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestDeleteService_NotFound(t *testing.T) {
	repo := newFakeServiceRepo()
	h := handlers.NewServiceHandler(repo)

	req := httptest.NewRequest(http.MethodDelete, "/api/services/nonexistent", nil)
	req = withChiParam(req, "id", "nonexistent")
	w := httptest.NewRecorder()

	h.Delete(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestDeleteService_Success(t *testing.T) {
	repo := newFakeServiceRepo()
	h := handlers.NewServiceHandler(repo)

	repo.services["to-delete"] = &models.Service{ID: "to-delete", Name: "Bye", URL: "https://bye.com"}

	req := httptest.NewRequest(http.MethodDelete, "/api/services/to-delete", nil)
	req = withChiParam(req, "id", "to-delete")
	w := httptest.NewRecorder()

	h.Delete(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}
	if len(repo.services) != 0 {
		t.Errorf("expected service to be deleted from repo")
	}
}
