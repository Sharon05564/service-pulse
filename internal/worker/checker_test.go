package worker_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sharonowusu/servicepulse/internal/models"
	"github.com/sharonowusu/servicepulse/internal/worker"
)

// TestProbe_Success verifies that a 200-returning server is recorded as healthy.
func TestProbe_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	svc := models.Service{
		ID:                 "svc-1",
		Name:               "test",
		URL:                ts.URL,
		Method:             "GET",
		ExpectedStatusCode: 200,
	}

	result := worker.Probe(svc)

	if !result.IsSuccess {
		t.Errorf("expected success, got failure: %v", result.ErrorMessage)
	}
	if result.ResponseTimeMs <= 0 {
		t.Errorf("expected positive latency, got %d", result.ResponseTimeMs)
	}
	if !result.StatusCode.Valid || result.StatusCode.Int64 != 200 {
		t.Errorf("expected status 200, got %v", result.StatusCode)
	}
}

// TestProbe_WrongStatus verifies that a mismatched status code is a failure.
func TestProbe_WrongStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable) // 503
	}))
	defer ts.Close()

	svc := models.Service{
		ID:                 "svc-2",
		Name:               "flaky",
		URL:                ts.URL,
		Method:             "GET",
		ExpectedStatusCode: 200,
	}

	result := worker.Probe(svc)

	if result.IsSuccess {
		t.Error("expected failure, got success")
	}
	if !result.ErrorMessage.Valid {
		t.Error("expected error message to be set")
	}
}

// TestProbe_NetworkError verifies that an unreachable host is a failure.
func TestProbe_NetworkError(t *testing.T) {
	svc := models.Service{
		ID:                 "svc-3",
		Name:               "dead",
		URL:                "http://127.0.0.1:1", // nothing listening here
		Method:             "GET",
		ExpectedStatusCode: 200,
	}

	result := worker.Probe(svc)

	if result.IsSuccess {
		t.Error("expected failure for unreachable host, got success")
	}
}
