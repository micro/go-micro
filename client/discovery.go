package client

import (
	"errors"
	"fmt"

	merrors "go-micro.dev/v6/errors"
	"go-micro.dev/v6/registry"
	"go-micro.dev/v6/selector"
)

// ErrNoNodes means discovery succeeded but no usable node is available.
var ErrNoNodes = selector.ErrNoneAvailable

// DiscoveryError distinguishes missing services from unavailable discovery.
// Unwrap retains the backend cause, and As exposes the structured RPC status.
type DiscoveryError struct {
	Service string
	Cause   error
	status  *merrors.Error
}

func (e *DiscoveryError) Error() string { return e.status.Error() }
func (e *DiscoveryError) Unwrap() error { return e.Cause }
func (e *DiscoveryError) As(target interface{}) bool {
	if out, ok := target.(**merrors.Error); ok {
		*out = e.status
		return true
	}
	return false
}
func (e *DiscoveryError) Is(target error) bool {
	return target == registry.ErrNotFound && (errors.Is(e.Cause, selector.ErrNotFound) || errors.Is(e.Cause, registry.ErrNotFound))
}

// NewDiscoveryError translates selection failures consistently for RPC clients.
func NewDiscoveryError(service string, cause error) error {
	if cause == nil {
		return nil
	}
	code := int32(503)
	if errors.Is(cause, selector.ErrNotFound) || errors.Is(cause, registry.ErrNotFound) {
		code = 404
	}
	return &DiscoveryError{Service: service, Cause: cause, status: merrors.FromError(merrors.New("go.micro.client", fmt.Sprintf("discover service %s: %v", service, cause), code))}
}
