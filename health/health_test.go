package health

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealth_AllUp(t *testing.T) {
	h := NewHandler()
	h.Register("db", func(ctx context.Context) error {
		return nil
	})

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	h.Health()(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp HealthResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Status != StatusUp {
		t.Errorf("expected UP, got %s", resp.Status)
	}
}

func TestHealth_Down(t *testing.T) {
	h := NewHandler()
	h.Register("db", func(ctx context.Context) error {
		return errors.New("connection failed")
	})

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	h.Health()(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}

	var resp HealthResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Status != StatusDown {
		t.Errorf("expected DOWN, got %s", resp.Status)
	}
}

func TestLiveness(t *testing.T) {
	h := NewHandler()
	
	req := httptest.NewRequest("GET", "/health/live", nil)
	w := httptest.NewRecorder()
	h.Liveness()(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestInfo(t *testing.T) {
	h := NewHandler(WithAppInfo("test-service", "1.0.0", "Test"))

	req := httptest.NewRequest("GET", "/info", nil)
	w := httptest.NewRecorder()
	h.Info()(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp InfoResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.App.Name != "test-service" {
		t.Errorf("expected test-service, got %s", resp.App.Name)
	}
}

func TestMetrics(t *testing.T) {
	h := NewHandler()

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	h.Metrics()(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "text/plain; version=0.0.4" {
		t.Errorf("expected prometheus content type, got %s", contentType)
	}
}
