---
layout: default
---

# Graceful Shutdown

Properly shutting down services to avoid dropped requests and data loss.

## The Problem

Without graceful shutdown:
- In-flight requests are dropped
- Database connections leak
- Resources aren't cleaned up
- Load balancers don't know service is down

## Solution

Go Micro handles SIGTERM/SIGINT by default, but you need to implement cleanup logic.

## Basic Pattern

```go
package main

import (
    "context"
    "os"
    "os/signal"
    "syscall"
    "time"
    "go-micro.dev/v5"
    "go-micro.dev/v5/logger"
)

func main() {
    svc := micro.NewService(
        micro.Name("myservice"),
        micro.BeforeStop(func() error {
            logger.Info("Service stopping, running cleanup...")
            return cleanup()
        }),
    )

    svc.Init()

    // Your service logic
    if err := svc.Handle(new(Handler)); err != nil {
        logger.Fatal(err)
    }

    // Run with graceful shutdown
    if err := svc.Run(); err != nil {
        logger.Fatal(err)
    }

    logger.Info("Service stopped gracefully")
}

func cleanup() error {
    // Close database connections
    // Flush logs
    // Stop background workers
    // etc.
    return nil
}
```

## Database Cleanup

```go
type Service struct {
    db *sql.DB
}

func (s *Service) Shutdown(ctx context.Context) error {
    logger.Info("Closing database connections...")
    
    // Stop accepting new requests
    s.db.SetMaxOpenConns(0)
    
    // Wait for existing connections to finish (with timeout)
    done := make(chan struct{})
    go func() {
        s.db.Close()
        close(done)
    }()
    
    select {
    case <-done:
        logger.Info("Database closed gracefully")
        return nil
    case <-ctx.Done():
        logger.Warn("Database close timeout, forcing")
        return ctx.Err()
    }
}
```

## Background Workers

```go
type Worker struct {
    quit chan struct{}
    done chan struct{}
}

func (w *Worker) Start() {
    w.quit = make(chan struct{})
    w.done = make(chan struct{})
    
    go func() {
        defer close(w.done)
        ticker := time.NewTicker(5 * time.Second)
        defer ticker.Stop()
        
        for {
            select {
            case <-ticker.C:
                w.doWork()
            case <-w.quit:
                logger.Info("Worker stopping...")
                return
            }
        }
    }()
}

func (w *Worker) Stop(timeout time.Duration) error {
    close(w.quit)
    
    select {
    case <-w.done:
        logger.Info("Worker stopped gracefully")
        return nil
    case <-time.After(timeout):
        return fmt.Errorf("worker shutdown timeout")
    }
}
```

## Complete Example

```go
package main

import (
    "context"
    "database/sql"
    "fmt"
    "os"
    "os/signal"
    "sync"
    "syscall"
    "time"
    
    "go-micro.dev/v5"
    "go-micro.dev/v5/logger"
)

type Application struct {
    db      *sql.DB
    workers []*Worker
    wg      sync.WaitGroup
    mu      sync.RWMutex
    closing bool
}

func NewApplication(db *sql.DB) *Application {
    return &Application{
        db:      db,
        workers: make([]*Worker, 0),
    }
}

func (app *Application) AddWorker(w *Worker) {
    app.workers = append(app.workers, w)
    w.Start()
}

func (app *Application) Shutdown(ctx context.Context) error {
    app.mu.Lock()
    if app.closing {
        app.mu.Unlock()
        return nil
    }
    app.closing = true
    app.mu.Unlock()
    
    logger.Info("Starting graceful shutdown...")
    
    // Stop accepting new work
    logger.Info("Stopping workers...")
    for _, w := range app.workers {
        if err := w.Stop(5 * time.Second); err != nil {
            logger.Warnf("Worker failed to stop: %v", err)
        }
    }
    
    // Wait for in-flight requests (with timeout)
    shutdownComplete := make(chan struct{})
    go func() {
        app.wg.Wait()
        close(shutdownComplete)
    }()
    
    select {
    case <-shutdownComplete:
        logger.Info("All requests completed")
    case <-ctx.Done():
        logger.Warn("Shutdown timeout, forcing...")
    }
    
    // Close resources
    logger.Info("Closing database...")
    if err := app.db.Close(); err != nil {
        logger.Errorf("Database close error: %v", err)
    }
    
    logger.Info("Shutdown complete")
    return nil
}

func main() {
    db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
    if err != nil {
        logger.Fatal(err)
    }
    
    app := NewApplication(db)
    
    // Add background workers
    app.AddWorker(&Worker{name: "cleanup"})
    app.AddWorker(&Worker{name: "metrics"})
    
    svc := micro.NewService(
        micro.Name("myservice"),
        micro.BeforeStop(func() error {
            ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
            defer cancel()
            return app.Shutdown(ctx)
        }),
    )
    
    svc.Init()
    
    handler := &Handler{app: app}
    if err := svc.Handle(handler); err != nil {
        logger.Fatal(err)
    }
    
    // Run service
    if err := svc.Run(); err != nil {
        logger.Fatal(err)
    }
}
```

## Kubernetes Integration

### Liveness and Readiness Probes

```go
func (h *Handler) Health(ctx context.Context, req *struct{}, rsp *HealthResponse) error {
    // Liveness: is the service alive?
    rsp.Status = "ok"
    return nil
}

func (h *Handler) Ready(ctx context.Context, req *struct{}, rsp *ReadyResponse) error {
    h.app.mu.RLock()
    closing := h.app.closing
    h.app.mu.RUnlock()
    
    if closing {
        // Stop receiving traffic during shutdown
        return fmt.Errorf("shutting down")
    }
    
    // Check dependencies
    if err := h.app.db.Ping(); err != nil {
        return fmt.Errorf("database unhealthy: %w", err)
    }
    
    rsp.Status = "ready"
    return nil
}
```

### Kubernetes Manifest

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: myservice
spec:
  replicas: 3
  template:
    spec:
      containers:
      - name: myservice
        image: myservice:latest
        ports:
        - containerPort: 8080
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
        lifecycle:
          preStop:
            exec:
              # Give service time to drain before SIGTERM
              command: ["/bin/sh", "-c", "sleep 10"]
      terminationGracePeriodSeconds: 40
```

## Best Practices

1. **Set timeouts**: Don't wait forever for shutdown
2. **Stop accepting work early**: Set readiness to false
3. **Drain in-flight requests**: Let current work finish
4. **Close resources properly**: Databases, file handles, etc.
5. **Log shutdown progress**: Help debugging
6. **Handle SIGTERM and SIGINT**: Kubernetes sends SIGTERM
7. **Coordinate with load balancer**: Use readiness probes
8. **Test shutdown**: Regularly test graceful shutdown works

## Testing Shutdown

```bash
# Start service
go run main.go &
PID=$!

# Send some requests
for i in {1..10}; do
    curl http://localhost:8080/endpoint &
done

# Trigger graceful shutdown
kill -TERM $PID

# Verify all requests completed
wait
```

## Common Pitfalls

- **No timeout**: Service hangs during shutdown
- **Not stopping workers**: Background jobs continue
- **Database leaks**: Connections not closed
- **Ignored signals**: Service killed forcefully
- **No readiness probe**: Traffic during shutdown

## Related

- [API Gateway Example](api-gateway.md) - Multi-service architecture
- [Getting Started Guide](../../getting-started.md) - Basic service setup
