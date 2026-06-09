package health

import (
	"context"
	"fmt"

	"go-micro.dev/v5/registry"
)

// RegistryCheck creates a health check that verifies connectivity to the
// service registry. It calls ListServices on the registry to confirm
// the connection is alive. This is useful for Kubernetes readiness probes
// to detect when a service has lost its connection to the registry (e.g. etcd).
//
// Usage:
//
//	health.Register("registry", health.RegistryCheck(reg))
func RegistryCheck(reg registry.Registry) CheckFunc {
	return func(ctx context.Context) error {
		if reg == nil {
			return fmt.Errorf("registry is nil")
		}
		_, err := reg.ListServices()
		if err != nil {
			return fmt.Errorf("registry %s: %w", reg.String(), err)
		}
		return nil
	}
}
