# OpenTelemetry wrappers

OpenTelemetry wrappers propagate traces (spans) accross services.

## Usage

```go
service := micro.NewService(
    micro.Name("go.micro.srv.greeter"),
    micro.WrapClient(opentelemetry.NewClientWrapper()),
    micro.WrapHandler(opentelemetry.NewHandlerWrapper()),
    micro.WrapSubscriber(opentelemetry.NewSubscriberWrapper()),
)
```