// Package api provides HTTP API gateway functionality for go-micro services.
//
// The API gateway translates HTTP requests into RPC calls and serves a web dashboard
// for browsing and calling services. It can be used in development (micro run) or
// production (micro server) with optional authentication.
package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"go-micro.dev/v5/gateway/mcp"
	"go-micro.dev/v5/registry"
)

// Options configures the HTTP API gateway
type Options struct {
	// Address to listen on (e.g., ":8080")
	Address string

	// AuthEnabled controls whether authentication is required
	// If true, the HandlerRegistrar should include auth middleware
	AuthEnabled bool

	// Context for cancellation (if nil, uses background context)
	Context context.Context

	// Logger for gateway messages (if nil, uses log.Default())
	Logger *log.Logger

	// HandlerRegistrar is called to register HTTP handlers on the mux
	// This allows different configurations (dev vs prod) to register different handlers
	HandlerRegistrar func(mux *http.ServeMux) error

	// MCPEnabled controls whether to start MCP gateway
	MCPEnabled bool

	// MCPAddress is the address for MCP gateway (e.g., ":3000")
	MCPAddress string

	// Registry for service discovery (if nil, uses registry.DefaultRegistry)
	Registry registry.Registry
}

// Gateway represents a running HTTP API gateway server
type Gateway struct {
	opts   Options
	server *http.Server
	mux    *http.ServeMux
}

// New creates a new gateway with the given options and starts it.
// Returns immediately after starting the server in a goroutine.
// Use Wait() or Run() to block until the server stops.
func New(opts Options) (*Gateway, error) {
	// Set defaults
	if opts.Address == "" {
		opts.Address = ":8080"
	}
	if opts.Context == nil {
		opts.Context = context.Background()
	}
	if opts.Logger == nil {
		opts.Logger = log.Default()
	}
	if opts.Registry == nil {
		opts.Registry = registry.DefaultRegistry
	}

	// Create a new mux for this gateway instance
	mux := http.NewServeMux()

	// Register handlers using the provided registrar
	if opts.HandlerRegistrar != nil {
		if err := opts.HandlerRegistrar(mux); err != nil {
			return nil, fmt.Errorf("failed to register handlers: %w", err)
		}
	}

	// Create HTTP server
	server := &http.Server{
		Addr:    opts.Address,
		Handler: mux,
	}

	gw := &Gateway{
		opts:   opts,
		server: server,
		mux:    mux,
	}

	// Start MCP gateway if enabled
	if opts.MCPEnabled && opts.MCPAddress != "" {
		go func() {
			if err := mcp.ListenAndServe(opts.MCPAddress, mcp.Options{
				Registry: opts.Registry,
				Context:  opts.Context,
				Logger:   opts.Logger,
			}); err != nil {
				opts.Logger.Printf("[mcp] MCP gateway error: %v", err)
			}
		}()
		opts.Logger.Printf("[mcp] MCP gateway enabled on %s", opts.MCPAddress)
	}

	// Start server in background
	go func() {
		opts.Logger.Printf("[gateway] Listening on %s (auth: %v)", opts.Address, opts.AuthEnabled)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			opts.Logger.Printf("[gateway] Server error: %v", err)
		}
	}()

	return gw, nil
}

// Run creates and starts a gateway, blocking until it stops.
// This is a convenience function equivalent to New() + Wait().
func Run(opts Options) error {
	gw, err := New(opts)
	if err != nil {
		return err
	}
	return gw.Wait()
}

// Wait blocks until the server is shut down
func (g *Gateway) Wait() error {
	<-g.opts.Context.Done()
	return g.Stop()
}

// Stop gracefully shuts down the gateway
func (g *Gateway) Stop() error {
	if g.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return g.server.Shutdown(ctx)
	}
	return nil
}

// Addr returns the address the gateway is listening on
func (g *Gateway) Addr() string {
	return g.opts.Address
}

// Mux returns the underlying HTTP mux for this gateway
// This can be used to register additional handlers after creation
func (g *Gateway) Mux() *http.ServeMux {
	return g.mux
}
