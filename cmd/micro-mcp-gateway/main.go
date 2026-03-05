// Command micro-mcp-gateway runs a standalone MCP gateway that discovers
// go-micro services via a registry and exposes them as AI-accessible tools
// through the Model Context Protocol.
//
// This is the production deployment binary for the MCP gateway, intended
// to run independently of your services.
//
// Usage:
//
//	# mDNS (development default)
//	micro-mcp-gateway --address :3000
//
//	# Consul
//	micro-mcp-gateway --address :3000 --registry consul --registry-address consul:8500
//
//	# etcd
//	micro-mcp-gateway --address :3000 --registry etcd --registry-address etcd:2379
//
//	# With auth and rate limiting
//	micro-mcp-gateway --address :3000 --registry consul \
//	    --rate-limit 100 --rate-burst 200 --audit
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"go-micro.dev/v5/auth"
	"go-micro.dev/v5/auth/jwt"
	"go-micro.dev/v5/gateway/mcp"
	"go-micro.dev/v5/registry"
	"go-micro.dev/v5/registry/consul"
	"go-micro.dev/v5/registry/etcd"

	"github.com/urfave/cli/v2"
)

var version = "0.1.0"

func main() {
	app := &cli.App{
		Name:    "micro-mcp-gateway",
		Usage:   "Standalone MCP gateway for go-micro services",
		Version: version,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "address",
				Usage:   "Address to listen on",
				Value:   ":3000",
				EnvVars: []string{"MCP_ADDRESS"},
			},
			&cli.StringFlag{
				Name:    "registry",
				Usage:   "Service registry (mdns, consul, etcd)",
				Value:   "mdns",
				EnvVars: []string{"MICRO_REGISTRY"},
			},
			&cli.StringFlag{
				Name:    "registry-address",
				Usage:   "Registry address (e.g., consul:8500, etcd:2379)",
				EnvVars: []string{"MICRO_REGISTRY_ADDRESS"},
			},
			&cli.Float64Flag{
				Name:    "rate-limit",
				Usage:   "Requests per second per tool (0 = unlimited)",
				EnvVars: []string{"MCP_RATE_LIMIT"},
			},
			&cli.IntFlag{
				Name:    "rate-burst",
				Usage:   "Rate limit burst size",
				Value:   20,
				EnvVars: []string{"MCP_RATE_BURST"},
			},
			&cli.BoolFlag{
				Name:    "auth",
				Usage:   "Enable JWT authentication",
				EnvVars: []string{"MCP_AUTH"},
			},
			&cli.BoolFlag{
				Name:    "audit",
				Usage:   "Enable audit logging to stdout",
				EnvVars: []string{"MCP_AUDIT"},
			},
			&cli.StringSliceFlag{
				Name:  "scope",
				Usage: "Tool scope requirement (format: tool=scope1,scope2)",
			},
			&cli.IntFlag{
				Name:    "circuit-breaker",
				Usage:   "Circuit breaker max failures before opening (0 = disabled)",
				EnvVars: []string{"MCP_CIRCUIT_BREAKER"},
			},
			&cli.DurationFlag{
				Name:    "circuit-breaker-timeout",
				Usage:   "Circuit breaker open-state timeout before half-open probe",
				Value:   30 * time.Second,
				EnvVars: []string{"MCP_CIRCUIT_BREAKER_TIMEOUT"},
			},
		},
		Action: run,
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func run(c *cli.Context) error {
	logger := log.New(os.Stdout, "[mcp-gateway] ", log.LstdFlags)

	// Configure registry
	reg, err := newRegistry(c.String("registry"), c.String("registry-address"))
	if err != nil {
		return fmt.Errorf("registry: %w", err)
	}

	// Build MCP options
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	opts := mcp.Options{
		Registry: reg,
		Address:  c.String("address"),
		Context:  ctx,
		Logger:   logger,
	}

	// Rate limiting
	if rps := c.Float64("rate-limit"); rps > 0 {
		opts.RateLimit = &mcp.RateLimitConfig{
			RequestsPerSecond: rps,
			Burst:             c.Int("rate-burst"),
		}
		logger.Printf("Rate limit: %.0f req/s, burst %d", rps, c.Int("rate-burst"))
	}

	// Auth
	if c.Bool("auth") {
		opts.Auth = jwt.NewAuth()
		logger.Printf("JWT authentication enabled")
	}

	// Scopes
	if scopes := c.StringSlice("scope"); len(scopes) > 0 {
		opts.Scopes = parseScopes(scopes)
		for tool, s := range opts.Scopes {
			logger.Printf("Scope: %s requires [%s]", tool, strings.Join(s, ", "))
		}
	}

	// Circuit breaker
	if maxFail := c.Int("circuit-breaker"); maxFail > 0 {
		opts.CircuitBreaker = &mcp.CircuitBreakerConfig{
			MaxFailures: maxFail,
			Timeout:     c.Duration("circuit-breaker-timeout"),
		}
		logger.Printf("Circuit breaker: max %d failures, timeout %s", maxFail, c.Duration("circuit-breaker-timeout"))
	}

	// Audit
	if c.Bool("audit") {
		opts.AuditFunc = func(r mcp.AuditRecord) {
			status := "ALLOWED"
			if !r.Allowed {
				status = "DENIED:" + r.DeniedReason
			}
			logger.Printf("[audit] %s tool=%s account=%s status=%s duration=%s",
				r.TraceID, r.Tool, r.AccountID, status, r.Duration)
		}
		logger.Printf("Audit logging enabled")
	}

	// Print startup info
	logger.Printf("Starting MCP gateway on %s", c.String("address"))
	logger.Printf("Registry: %s", c.String("registry"))
	if addr := c.String("registry-address"); addr != "" {
		logger.Printf("Registry address: %s", addr)
	}

	// Start gateway in background
	errCh := make(chan error, 1)
	go func() {
		errCh <- mcp.ListenAndServe(opts.Address, opts)
	}()

	// Wait for signal or error
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		logger.Printf("Received %s, shutting down...", sig)
		cancel()
		return nil
	case err := <-errCh:
		return fmt.Errorf("gateway error: %w", err)
	}
}

func newRegistry(name, address string) (registry.Registry, error) {
	var opts []registry.Option
	if address != "" {
		opts = append(opts, registry.Addrs(strings.Split(address, ",")...))
	}

	switch name {
	case "mdns", "":
		return registry.NewMDNSRegistry(opts...), nil
	case "consul":
		return consul.NewConsulRegistry(opts...), nil
	case "etcd":
		return etcd.NewEtcdRegistry(opts...), nil
	default:
		return nil, fmt.Errorf("unknown registry %q (supported: mdns, consul, etcd)", name)
	}
}

func parseScopes(raw []string) map[string][]string {
	scopes := make(map[string][]string)
	for _, s := range raw {
		parts := strings.SplitN(s, "=", 2)
		if len(parts) != 2 {
			continue
		}
		tool := strings.TrimSpace(parts[0])
		scopeList := strings.Split(parts[1], ",")
		for i := range scopeList {
			scopeList[i] = strings.TrimSpace(scopeList[i])
		}
		scopes[tool] = scopeList
	}
	return scopes
}

// Ensure auth.Auth interface is satisfied at compile time.
var _ auth.Auth = jwt.NewAuth()
