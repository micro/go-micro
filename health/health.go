// Package health provides health checks and metrics for go-micro services.
// Similar to Spring Boot Actuator.
package health

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"sync"
	"time"
)

// Status represents the health status.
type Status string

const (
	StatusUp      Status = "UP"
	StatusDown    Status = "DOWN"
	StatusUnknown Status = "UNKNOWN"
)

// Check is a health check function.
type Check func(ctx context.Context) error

// CheckResult represents the result of a health check.
type CheckResult struct {
	Status  Status                 `json:"status"`
	Details map[string]interface{} `json:"details,omitempty"`
	Error   string                 `json:"error,omitempty"`
}

// HealthResponse is the response for the health endpoint.
type HealthResponse struct {
	Status Status                  `json:"status"`
	Checks map[string]*CheckResult `json:"checks,omitempty"`
}

// InfoResponse is the response for the info endpoint.
type InfoResponse struct {
	App     AppInfo     `json:"app"`
	Runtime RuntimeInfo `json:"runtime"`
}

// AppInfo contains application information.
type AppInfo struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description,omitempty"`
}

// RuntimeInfo contains runtime information.
type RuntimeInfo struct {
	GoVersion    string `json:"go_version"`
	NumCPU       int    `json:"num_cpu"`
	NumGoroutine int    `json:"num_goroutine"`
	MemAlloc     uint64 `json:"mem_alloc_bytes"`
	MemTotal     uint64 `json:"mem_total_bytes"`
	Uptime       string `json:"uptime"`
}

// Handler provides health check HTTP handlers.
type Handler struct {
	checks    map[string]Check
	appInfo   AppInfo
	startTime time.Time
	mu        sync.RWMutex
}

// Option configures the health handler.
type Option func(*Handler)

// WithAppInfo sets application information.
func WithAppInfo(name, version, description string) Option {
	return func(h *Handler) {
		h.appInfo = AppInfo{
			Name:        name,
			Version:     version,
			Description: description,
		}
	}
}

// NewHandler creates a new health handler.
func NewHandler(opts ...Option) *Handler {
	h := &Handler{
		checks:    make(map[string]Check),
		startTime: time.Now(),
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// Register registers a health check.
func (h *Handler) Register(name string, check Check) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.checks[name] = check
}

// Health returns the health check HTTP handler.
func (h *Handler) Health() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		h.mu.RLock()
		checks := make(map[string]Check, len(h.checks))
		for k, v := range h.checks {
			checks[k] = v
		}
		h.mu.RUnlock()

		response := HealthResponse{
			Status: StatusUp,
			Checks: make(map[string]*CheckResult),
		}

		for name, check := range checks {
			result := &CheckResult{Status: StatusUp}
			if err := check(ctx); err != nil {
				result.Status = StatusDown
				result.Error = err.Error()
				response.Status = StatusDown
			}
			response.Checks[name] = result
		}

		status := http.StatusOK
		if response.Status == StatusDown {
			status = http.StatusServiceUnavailable
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(response)
	}
}

// Liveness returns a simple liveness probe handler.
func (h *Handler) Liveness() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "UP"})
	}
}

// Readiness returns a readiness probe handler.
func (h *Handler) Readiness() http.HandlerFunc {
	return h.Health()
}

// Info returns the info endpoint handler.
func (h *Handler) Info() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		response := InfoResponse{
			App: h.appInfo,
			Runtime: RuntimeInfo{
				GoVersion:    runtime.Version(),
				NumCPU:       runtime.NumCPU(),
				NumGoroutine: runtime.NumGoroutine(),
				MemAlloc:     m.Alloc,
				MemTotal:     m.TotalAlloc,
				Uptime:       time.Since(h.startTime).Round(time.Second).String(),
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// Metrics returns a basic metrics endpoint handler.
// For production, integrate with Prometheus using promhttp.
func (h *Handler) Metrics() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		// Prometheus-compatible text format
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		
		metrics := fmt.Sprintf(`# HELP go_goroutines Number of goroutines
# TYPE go_goroutines gauge
go_goroutines %d
# HELP go_memstats_alloc_bytes Number of bytes allocated
# TYPE go_memstats_alloc_bytes gauge
go_memstats_alloc_bytes %d
# HELP go_memstats_heap_objects Number of heap objects
# TYPE go_memstats_heap_objects gauge
go_memstats_heap_objects %d
# HELP process_uptime_seconds Time since process start
# TYPE process_uptime_seconds counter
process_uptime_seconds %f
`, runtime.NumGoroutine(), m.Alloc, m.HeapObjects, time.Since(h.startTime).Seconds())
		
		w.Write([]byte(metrics))
	}
}

// Common health checks

// DatabaseCheck returns a health check for a database.
func DatabaseCheck(pingFn func(ctx context.Context) error) Check {
	return func(ctx context.Context) error {
		return pingFn(ctx)
	}
}

// HTTPCheck returns a health check that calls an HTTP endpoint.
func HTTPCheck(url string) Check {
	return func(ctx context.Context) error {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return err
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 400 {
			return fmt.Errorf("health check failed: status %d", resp.StatusCode)
		}
		return nil
	}
}

// Default handler instance
var defaultHandler = NewHandler()

// Register registers a health check on the default handler.
func Register(name string, check Check) {
	defaultHandler.Register(name, check)
}

// Health returns the health handler.
func Health() http.HandlerFunc {
	return defaultHandler.Health()
}

// Liveness returns the liveness handler.
func Liveness() http.HandlerFunc {
	return defaultHandler.Liveness()
}

// Readiness returns the readiness handler.
func Readiness() http.HandlerFunc {
	return defaultHandler.Readiness()
}

// Info returns the info handler.
func Info() http.HandlerFunc {
	return defaultHandler.Info()
}

// Metrics returns the metrics handler.
func Metrics() http.HandlerFunc {
	return defaultHandler.Metrics()
}
