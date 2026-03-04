package service

import (
	"context"
	"os"
	"os/signal"
	"sync"

	log "go-micro.dev/v5/logger"
	signalutil "go-micro.dev/v5/util/signal"
)

// Group runs multiple services in a single binary with shared
// lifecycle management. All services start together and stop
// together on signal or context cancellation.
type Group struct {
	services []Service
	logger   log.Logger
}

// NewGroup creates a new service group.
func NewGroup(svcs ...Service) *Group {
	return &Group{
		services: svcs,
		logger:   log.DefaultLogger,
	}
}

// Add appends one or more services to the group.
func (g *Group) Add(svcs ...Service) {
	g.services = append(g.services, svcs...)
}

// Run starts all services concurrently and blocks until a signal
// is received or the context is cancelled, then stops all services.
func (g *Group) Run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize all services. Disable per-service signal handling
	// since the group manages signals.
	for _, svc := range g.services {
		svc.Init(HandleSignal(false))
	}

	g.logger.Logf(log.InfoLevel, "Starting service group with %d services", len(g.services))

	// Start all services
	errCh := make(chan error, len(g.services))
	for _, svc := range g.services {
		g.logger.Logf(log.InfoLevel, "Starting [service] %s", svc.Name())
		if err := svc.Start(); err != nil {
			cancel()
			g.stopAll()
			return err
		}
	}

	// Wait for signal or context cancellation
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, signalutil.Shutdown()...)

	select {
	case <-ch:
		g.logger.Logf(log.InfoLevel, "Received signal, stopping all services")
	case <-ctx.Done():
	case err := <-errCh:
		cancel()
		g.stopAll()
		return err
	}

	return g.stopAll()
}

func (g *Group) stopAll() error {
	var (
		mu      sync.Mutex
		lastErr error
	)

	var wg sync.WaitGroup
	for _, svc := range g.services {
		wg.Add(1)
		go func(s Service) {
			defer wg.Done()
			g.logger.Logf(log.InfoLevel, "Stopping [service] %s", s.Name())
			if err := s.Stop(); err != nil {
				mu.Lock()
				lastErr = err
				mu.Unlock()
			}
		}(svc)
	}
	wg.Wait()

	return lastErr
}
