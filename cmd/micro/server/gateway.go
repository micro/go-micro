package server

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"go-micro.dev/v5/gateway/api"
	"go-micro.dev/v5/registry"
	"go-micro.dev/v5/store"
)

// GatewayOptions configures the HTTP gateway (legacy compatibility)
// Deprecated: Use gateway/api.Options directly
type GatewayOptions = api.Options

// Gateway represents a running HTTP gateway server (legacy compatibility)
// Deprecated: Use gateway/api.Gateway directly
type Gateway = api.Gateway

// StartGateway starts the HTTP gateway with the given options.
// This is a compatibility wrapper around gateway/api.New().
//
// Deprecated: Use gateway/api.New() directly for new code.
func StartGateway(opts GatewayOptions) (*Gateway, error) {
	// Initialize auth if enabled (server-specific setup)
	if opts.AuthEnabled {
		if err := initAuth(); err != nil {
			return nil, fmt.Errorf("failed to initialize auth: %w", err)
		}

		homeDir, _ := os.UserHomeDir()
		keyDir := filepath.Join(homeDir, "micro", "keys")
		privPath := filepath.Join(keyDir, "private.pem")
		pubPath := filepath.Join(keyDir, "public.pem")
		if err := InitJWTKeys(privPath, pubPath); err != nil {
			return nil, fmt.Errorf("failed to init JWT keys: %w", err)
		}
	}

	// Get store (server-specific default)
	s := store.DefaultStore

	// Parse templates (server-specific)
	tmpls := parseTemplates()

	// Create handler registrar that registers server-specific handlers
	opts.HandlerRegistrar = func(mux *http.ServeMux) error {
		registerHandlers(mux, tmpls, s, opts.AuthEnabled)
		return nil
	}

	// Use default registry if not set
	if opts.Registry == nil {
		opts.Registry = registry.DefaultRegistry
	}

	// Delegate to gateway/api package
	return api.New(opts)
}

// RunGateway starts the gateway and blocks until it stops.
//
// Deprecated: Use gateway/api.Run() with a custom handler registrar.
func RunGateway(opts GatewayOptions) error {
	gw, err := StartGateway(opts)
	if err != nil {
		return err
	}
	return gw.Wait()
}
