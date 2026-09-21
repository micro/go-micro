package micro

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go-micro.dev/v6/registry"
)

// WaitOptions controls dependency discovery and an optional readiness probe.
type WaitOptions struct {
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
	Probe          func(context.Context) error
}

// WaitOption configures WaitForService.
type WaitOption func(*WaitOptions)

// WaitBackoff sets the initial and maximum exponential retry delays.
func WaitBackoff(initial, maximum time.Duration) WaitOption {
	return func(o *WaitOptions) { o.InitialBackoff = initial; o.MaxBackoff = maximum }
}

// WaitProbe adds an application readiness check after discovery succeeds.
// The probe should be safe to repeat and honor its context, for example a read-only RPC.
func WaitProbe(probe func(context.Context) error) WaitOption {
	return func(o *WaitOptions) { o.Probe = probe }
}

// WaitForService waits for at least one registered node with an address, then
// runs the optional probe. Discovery alone does not guarantee RPC readiness.
// Retries use capped exponential backoff and stop when ctx is done. A registry
// backend or probe that ignores context may finish its in-flight attempt after
// this function returns; no additional attempts are started after cancellation.
func WaitForService(ctx context.Context, svc Service, name string, opts ...WaitOption) error {
	if svc == nil || svc.Options().Registry == nil || name == "" {
		return errors.New("wait for service: service, registry and dependency name are required")
	}
	options := WaitOptions{InitialBackoff: 100 * time.Millisecond, MaxBackoff: 2 * time.Second}
	for _, opt := range opts {
		opt(&options)
	}
	if options.InitialBackoff <= 0 || options.MaxBackoff < options.InitialBackoff {
		return errors.New("wait for service: backoff must be positive and maximum must not be smaller than initial")
	}
	reg := svc.Options().Registry
	delay := options.InitialBackoff
	var lastErr error
	for {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("wait for service %s: %w", name, errors.Join(err, lastErr))
		}
		result := make(chan error, 1)
		go func() {
			if err := ctx.Err(); err != nil {
				result <- err
				return
			}
			services, err := reg.GetService(name, func(o *registry.GetOptions) { o.Context = ctx })
			if err == nil {
				found := false
				for _, service := range services {
					if service == nil {
						continue
					}
					for _, node := range service.Nodes {
						if node != nil && node.Address != "" {
							found = true
							break
						}
					}
				}
				if !found {
					err = registry.ErrNotFound
				} else if options.Probe != nil {
					if err = ctx.Err(); err == nil {
						err = options.Probe(ctx)
					}
				}
			}
			result <- err
		}()
		select {
		case <-ctx.Done():
			return fmt.Errorf("wait for service %s: %w", name, errors.Join(ctx.Err(), lastErr))
		case lastErr = <-result:
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("wait for service %s: %w", name, errors.Join(err, lastErr))
			}
			if lastErr == nil {
				return nil
			}
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return fmt.Errorf("wait for service %s: %w", name, errors.Join(ctx.Err(), lastErr))
		case <-timer.C:
		}
		if delay > options.MaxBackoff/2 {
			delay = options.MaxBackoff
		} else {
			delay *= 2
		}
	}
}
