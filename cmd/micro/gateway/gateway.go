package gateway

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"go-micro.dev/v6/gateway/api"
	"go-micro.dev/v6/registry"
	"go-micro.dev/v6/store"
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
		if err := initGatewayAuth(); err != nil {
			return nil, err
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

func initGatewayAuth() error {
	if err := initAuth(); err != nil {
		return fmt.Errorf("initialize auth: %w", err)
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	keyDir := filepath.Join(homeDir, "micro", "keys")
	if err := InitJWTKeys(filepath.Join(keyDir, "private.pem"), filepath.Join(keyDir, "public.pem")); err != nil {
		return fmt.Errorf("initialize JWT keys: %w", err)
	}
	return nil
}
