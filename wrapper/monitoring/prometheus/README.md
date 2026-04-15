# Prometheus Wrapper

The `prometheus` wrapper package exposes standard request metrics (request
count, latency, errors) for go-micro services and clients, so they can be
scraped by a Prometheus server with zero extra boilerplate.

Resolves [micro/go-micro#2893](https://github.com/micro/go-micro/issues/2893).

## Installation

```go
import prom "go-micro.dev/v5/wrapper/monitoring/prometheus"
```

## Exported Metrics

All metrics are labelled with `service`, `endpoint` and `status`
(`"success"` or `"fail"`). Labels are kept small on purpose to avoid
blowing up Prometheus memory.

| Metric                          | Type      | Description                                 |
|---------------------------------|-----------|---------------------------------------------|
| `micro_request_total`           | Counter   | Total number of requests handled.           |
| `micro_request_duration_seconds`| Histogram | Request latency distribution (seconds).     |

The `micro` prefix can be overridden with `prom.ServiceName("myapp")`.

## Basic Usage

```go
import (
    "go-micro.dev/v5"
    prom "go-micro.dev/v5/wrapper/monitoring/prometheus"
)

func main() {
    service := micro.NewService(
        micro.Name("example.service"),
        micro.WrapHandler(prom.NewHandlerWrapper()),
        micro.WrapClient(prom.NewClientWrapper()),
        micro.WrapSubscriber(prom.NewSubscriberWrapper()),
    )

    service.Init()

    if err := service.Run(); err != nil {
        panic(err)
    }
}
```

To expose the metrics to Prometheus, serve the default `promhttp` handler
on a side HTTP endpoint:

```go
import (
    "net/http"

    "github.com/prometheus/client_golang/prometheus/promhttp"
)

go func() {
    http.Handle("/metrics", promhttp.Handler())
    _ = http.ListenAndServe(":9100", nil)
}()
```

Then point Prometheus at it:

```yaml
scrape_configs:
  - job_name: 'example.service'
    static_configs:
      - targets: ['localhost:9100']
```

## Wrappers

| Constructor               | Wraps                   | Notes                                      |
|---------------------------|-------------------------|--------------------------------------------|
| `NewHandlerWrapper`       | `server.HandlerWrapper` | Incoming RPC handlers.                     |
| `NewSubscriberWrapper`    | `server.SubscriberWrapper` | Event subscribers (uses topic as endpoint). |
| `NewCallWrapper`          | `client.CallWrapper`    | Outgoing unary RPC calls only.             |
| `NewClientWrapper`        | `client.Wrapper`        | Outgoing `Call` **and** `Publish`.         |

`NewClientWrapper` is the right choice when you want metrics for both
`Call` and `Publish`; use `NewCallWrapper` if you only care about unary
calls and want lower overhead.

## Configuration

All constructors accept functional options:

```go
prom.NewHandlerWrapper(
    prom.ServiceName("myapp"),                          // metric name prefix
    prom.Namespace("prod"),                             // Prometheus namespace
    prom.Subsystem("api"),                              // Prometheus subsystem
    prom.ConstLabels(prometheus.Labels{"dc": "eu-1"}),  // labels on every metric
    prom.Buckets([]float64{0.005, 0.05, 0.5, 1, 5}),    // latency buckets
    prom.Registerer(myRegistry),                        // custom registerer
)
```

Defaults:

- `ServiceName`: `"micro"`
- `Buckets`: `prometheus.DefBuckets`
- `Registerer`: `prometheus.DefaultRegisterer`

## Reusing Collectors

Creating multiple wrappers with the same options (e.g. `NewHandlerWrapper`
and `NewClientWrapper` together) is safe: the collectors are cached per
`(name, namespace, subsystem)` triple and `AlreadyRegisteredError` from
Prometheus is handled transparently, so the existing collector is reused.

## Testing

The package ships with unit tests that use a fresh `prometheus.Registry`
per test to keep assertions isolated:

```bash
go test ./wrapper/monitoring/prometheus/...
```

## License

Apache 2.0
