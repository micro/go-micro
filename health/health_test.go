package health

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRegisterAndRun(t *testing.T) {
	Reset()

	// Register a passing check
	Register("passing", func(ctx context.Context) error {
		return nil
	})

	resp := Run(context.Background())

	if resp.Status != StatusUp {
		t.Errorf("expected status up, got %s", resp.Status)
	}
	if len(resp.Checks) != 1 {
		t.Errorf("expected 1 check, got %d", len(resp.Checks))
	}
	if resp.Checks[0].Status != StatusUp {
		t.Errorf("expected check status up, got %s", resp.Checks[0].Status)
	}
}

func TestFailingCheck(t *testing.T) {
	Reset()

	Register("failing", func(ctx context.Context) error {
		return errors.New("database connection failed")
	})

	resp := Run(context.Background())

	if resp.Status != StatusDown {
		t.Errorf("expected status down, got %s", resp.Status)
	}
	if resp.Checks[0].Error != "database connection failed" {
		t.Errorf("expected error message, got %s", resp.Checks[0].Error)
	}
}

func TestNonCriticalCheck(t *testing.T) {
	Reset()

	// Register a non-critical failing check
	RegisterCheck(Check{
		Name: "optional",
		Check: func(ctx context.Context) error {
			return errors.New("optional service unavailable")
		},
		Critical: false,
	})

	resp := Run(context.Background())

	// Overall status should be up because check is not critical
	if resp.Status != StatusUp {
		t.Errorf("expected status up for non-critical failure, got %s", resp.Status)
	}
	// But the check itself should show as down
	if resp.Checks[0].Status != StatusDown {
		t.Errorf("expected check status down, got %s", resp.Checks[0].Status)
	}
}

func TestCheckTimeout(t *testing.T) {
	Reset()

	RegisterCheck(Check{
		Name: "slow",
		Check: func(ctx context.Context) error {
			select {
			case <-time.After(5 * time.Second):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
		Timeout:  100 * time.Millisecond,
		Critical: true,
	})

	resp := Run(context.Background())

	if resp.Status != StatusDown {
		t.Errorf("expected status down due to timeout, got %s", resp.Status)
	}
	if resp.Checks[0].Duration < 100*time.Millisecond {
		t.Errorf("expected duration >= 100ms, got %v", resp.Checks[0].Duration)
	}
}

func TestHealthHandler(t *testing.T) {
	Reset()

	Register("test", func(ctx context.Context) error {
		return nil
	})

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp.Status != StatusUp {
		t.Errorf("expected status up, got %s", resp.Status)
	}
}

func TestHealthHandlerUnhealthy(t *testing.T) {
	Reset()

	Register("failing", func(ctx context.Context) error {
		return errors.New("unhealthy")
	})

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	Handler().ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", w.Code)
	}
}

func TestLiveHandler(t *testing.T) {
	Reset()

	req := httptest.NewRequest("GET", "/health/live", nil)
	w := httptest.NewRecorder()

	LiveHandler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestReadyHandler(t *testing.T) {
	Reset()

	Register("db", func(ctx context.Context) error {
		return nil
	})

	req := httptest.NewRequest("GET", "/health/ready", nil)
	w := httptest.NewRecorder()

	ReadyHandler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestSetInfo(t *testing.T) {
	Reset()

	SetInfo("version", "1.0.0")
	SetInfo("service", "test-service")

	resp := Run(context.Background())

	if resp.Info["version"] != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", resp.Info["version"])
	}
	if resp.Info["service"] != "test-service" {
		t.Errorf("expected service test-service, got %s", resp.Info["service"])
	}
	// Should also have runtime info
	if resp.Info["go_version"] == "" {
		t.Error("expected go_version in info")
	}
}

func TestPingCheck(t *testing.T) {
	Reset()

	called := false
	Register("ping", PingCheck(func() error {
		called = true
		return nil
	}))

	Run(context.Background())

	if !called {
		t.Error("ping function was not called")
	}
}

func TestTCPCheck(t *testing.T) {
	// Start a TCP listener
	ln, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("failed to start listener: %v", err)
	}
	defer ln.Close()

	Reset()

	Register("tcp", TCPCheck(ln.Addr().String(), time.Second))

	resp := Run(context.Background())

	if resp.Status != StatusUp {
		t.Errorf("expected status up, got %s", resp.Status)
	}
}

func TestTCPCheckFailing(t *testing.T) {
	Reset()

	// Use a port that's unlikely to be listening
	Register("tcp", TCPCheck("localhost:59999", 100*time.Millisecond))

	resp := Run(context.Background())

	if resp.Status != StatusDown {
		t.Errorf("expected status down, got %s", resp.Status)
	}
}

func TestHTTPCheck(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer server.Close()

	Reset()

	Register("http", HTTPCheck(server.URL, time.Second))

	resp := Run(context.Background())

	if resp.Status != StatusUp {
		t.Errorf("expected status up, got %s", resp.Status)
	}
}

func TestHTTPCheckFailing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer server.Close()

	Reset()

	Register("http", HTTPCheck(server.URL, time.Second))

	resp := Run(context.Background())

	if resp.Status != StatusDown {
		t.Errorf("expected status down, got %s", resp.Status)
	}
}

func TestDNSCheck(t *testing.T) {
	Reset()

	Register("dns", DNSCheck("localhost"))

	resp := Run(context.Background())

	if resp.Status != StatusUp {
		t.Errorf("expected status up, got %s", resp.Status)
	}
}

func TestMultipleChecks(t *testing.T) {
	Reset()

	Register("check1", func(ctx context.Context) error { return nil })
	Register("check2", func(ctx context.Context) error { return nil })
	Register("check3", func(ctx context.Context) error { return nil })

	resp := Run(context.Background())

	if len(resp.Checks) != 3 {
		t.Errorf("expected 3 checks, got %d", len(resp.Checks))
	}
	if resp.Status != StatusUp {
		t.Errorf("expected status up, got %s", resp.Status)
	}
}

func TestRegisterHandlers(t *testing.T) {
	Reset()

	mux := http.NewServeMux()
	RegisterHandlers(mux)

	// Test /health
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("/health: expected 200, got %d", w.Code)
	}

	// Test /health/live
	req = httptest.NewRequest("GET", "/health/live", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("/health/live: expected 200, got %d", w.Code)
	}

	// Test /health/ready
	req = httptest.NewRequest("GET", "/health/ready", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("/health/ready: expected 200, got %d", w.Code)
	}
}

func TestIsReady(t *testing.T) {
	Reset()

	Register("check", func(ctx context.Context) error { return nil })

	if !IsReady(context.Background()) {
		t.Error("expected IsReady to return true")
	}

	Reset()

	Register("check", func(ctx context.Context) error { return errors.New("fail") })

	if IsReady(context.Background()) {
		t.Error("expected IsReady to return false")
	}
}

func TestConcurrentChecks(t *testing.T) {
	Reset()

	// Register multiple slow checks
	for i := 0; i < 5; i++ {
		Register("check"+string(rune('0'+i)), func(ctx context.Context) error {
			time.Sleep(50 * time.Millisecond)
			return nil
		})
	}

	start := time.Now()
	resp := Run(context.Background())
	duration := time.Since(start)

	// All checks run concurrently, should take ~50ms not ~250ms
	if duration > 150*time.Millisecond {
		t.Errorf("checks should run concurrently, took %v", duration)
	}

	if resp.Status != StatusUp {
		t.Errorf("expected status up, got %s", resp.Status)
	}
}
