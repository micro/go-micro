// Package health provides health check functionality for microservices.
//
// It supports Kubernetes-style liveness and readiness probes, along with
// pluggable health checks for dependencies like databases, caches, and
// external services.
//
// Basic usage:
//
//	// Register checks
//	health.Register("database", health.PingCheck(db.Ping))
//	health.Register("redis", health.TCPCheck("localhost:6379", time.Second))
//
//	// Add handlers
//	http.Handle("/health", health.Handler())
//	http.Handle("/health/live", health.LiveHandler())
//	http.Handle("/health/ready", health.ReadyHandler())
//
// Or use the convenience function to register all routes:
//
//	health.RegisterHandlers(mux)
package health

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"runtime"
	"sync"
	"time"
)

// Status represents the health status of a check or the overall system
type Status string

const (
	StatusUp   Status = "up"
	StatusDown Status = "down"
)

// CheckFunc is a function that performs a health check.
// It should return nil if healthy, or an error describing the problem.
type CheckFunc func(ctx context.Context) error

// Check represents a registered health check
type Check struct {
	Name     string
	Check    CheckFunc
	Timeout  time.Duration
	Critical bool // If true, failure marks the service as not ready
}

// Result represents the result of a health check
type Result struct {
	Name     string        `json:"name"`
	Status   Status        `json:"status"`
	Error    string        `json:"error,omitempty"`
	Duration time.Duration `json:"duration"`
}

// Response represents the overall health response
type Response struct {
	Status Status            `json:"status"`
	Checks []Result          `json:"checks,omitempty"`
	Info   map[string]string `json:"info,omitempty"`
}

var (
	mu             sync.RWMutex
	checks         []Check
	info           = make(map[string]string)
	defaultTimeout = 5 * time.Second
)

// Register adds a health check with default settings (critical, 5s timeout)
func Register(name string, check CheckFunc) {
	RegisterCheck(Check{
		Name:     name,
		Check:    check,
		Timeout:  defaultTimeout,
		Critical: true,
	})
}

// RegisterCheck adds a health check with custom settings
func RegisterCheck(check Check) {
	if check.Timeout == 0 {
		check.Timeout = defaultTimeout
	}
	mu.Lock()
	checks = append(checks, check)
	mu.Unlock()
}

// SetInfo sets metadata to include in health responses
func SetInfo(key, value string) {
	mu.Lock()
	info[key] = value
	mu.Unlock()
}

// Reset clears all registered checks and info (useful for testing)
func Reset() {
	mu.Lock()
	checks = nil
	info = make(map[string]string)
	mu.Unlock()
}

// Run executes all health checks and returns the results
func Run(ctx context.Context) Response {
	mu.RLock()
	checksCopy := make([]Check, len(checks))
	copy(checksCopy, checks)
	infoCopy := make(map[string]string)
	for k, v := range info {
		infoCopy[k] = v
	}
	mu.RUnlock()

	// Add runtime info
	infoCopy["go_version"] = runtime.Version()
	infoCopy["go_os"] = runtime.GOOS
	infoCopy["go_arch"] = runtime.GOARCH

	if len(checksCopy) == 0 {
		return Response{
			Status: StatusUp,
			Info:   infoCopy,
		}
	}

	// Run checks concurrently
	results := make([]Result, len(checksCopy))
	var wg sync.WaitGroup

	for i, check := range checksCopy {
		wg.Add(1)
		go func(i int, check Check) {
			defer wg.Done()
			results[i] = runCheck(ctx, check)
		}(i, check)
	}

	wg.Wait()

	// Determine overall status
	overallStatus := StatusUp
	for i, result := range results {
		if result.Status == StatusDown && checksCopy[i].Critical {
			overallStatus = StatusDown
			break
		}
	}

	return Response{
		Status: overallStatus,
		Checks: results,
		Info:   infoCopy,
	}
}

func runCheck(ctx context.Context, check Check) Result {
	ctx, cancel := context.WithTimeout(ctx, check.Timeout)
	defer cancel()

	start := time.Now()
	err := check.Check(ctx)
	duration := time.Since(start)

	result := Result{
		Name:     check.Name,
		Status:   StatusUp,
		Duration: duration,
	}

	if err != nil {
		result.Status = StatusDown
		result.Error = err.Error()
	}

	return result
}

// IsReady returns true if all critical checks pass
func IsReady(ctx context.Context) bool {
	resp := Run(ctx)
	return resp.Status == StatusUp
}

// IsLive always returns true (basic liveness)
// Override with SetLivenessCheck for custom behavior
func IsLive() bool {
	return true
}

// Handler returns an http.Handler for the main health endpoint
// Returns 200 if healthy, 503 if unhealthy
func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := Run(r.Context())
		writeResponse(w, resp)
	})
}

// LiveHandler returns an http.Handler for the liveness probe
// Returns 200 if the service is alive (basic check)
func LiveHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if IsLive() {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"up"}`))
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"down"}`))
		}
	})
}

// ReadyHandler returns an http.Handler for the readiness probe
// Returns 200 if all critical checks pass, 503 otherwise
func ReadyHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := Run(r.Context())
		writeResponse(w, resp)
	})
}

func writeResponse(w http.ResponseWriter, resp Response) {
	w.Header().Set("Content-Type", "application/json")
	if resp.Status == StatusUp {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	json.NewEncoder(w).Encode(resp)
}

// RegisterHandlers registers all health endpoints on the given mux
func RegisterHandlers(mux *http.ServeMux) {
	mux.Handle("/health", Handler())
	mux.Handle("/health/live", LiveHandler())
	mux.Handle("/health/ready", ReadyHandler())
}

// --- Built-in Checks ---

// PingCheck creates a check from a ping function (like sql.DB.Ping)
func PingCheck(ping func() error) CheckFunc {
	return func(ctx context.Context) error {
		return ping()
	}
}

// PingContextCheck creates a check from a ping function that accepts context
func PingContextCheck(ping func(context.Context) error) CheckFunc {
	return ping
}

// TCPCheck creates a check that verifies TCP connectivity
func TCPCheck(addr string, timeout time.Duration) CheckFunc {
	return func(ctx context.Context) error {
		conn, err := net.DialTimeout("tcp", addr, timeout)
		if err != nil {
			return fmt.Errorf("tcp dial %s: %w", addr, err)
		}
		conn.Close()
		return nil
	}
}

// HTTPCheck creates a check that verifies an HTTP endpoint returns 200
func HTTPCheck(url string, timeout time.Duration) CheckFunc {
	return func(ctx context.Context) error {
		client := &http.Client{Timeout: timeout}
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("http get %s: %w", url, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("http get %s: status %d", url, resp.StatusCode)
		}
		return nil
	}
}

// DNSCheck creates a check that verifies DNS resolution
func DNSCheck(host string) CheckFunc {
	return func(ctx context.Context) error {
		_, err := net.LookupHost(host)
		if err != nil {
			return fmt.Errorf("dns lookup %s: %w", host, err)
		}
		return nil
	}
}

// CustomCheck creates a check from any function returning an error
func CustomCheck(fn func() error) CheckFunc {
	return func(ctx context.Context) error {
		return fn()
	}
}
