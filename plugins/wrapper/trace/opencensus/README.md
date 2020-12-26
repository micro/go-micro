# OpenCensus wrappers

OpenCensus wrappers propagate traces (spans) accross services.

## Usage

```go
service := micro.NewService(
    micro.Name("go.micro.srv.greeter"),
    micro.WrapClient(opencensus.NewClientWrapper()),
    micro.WrapHandler(opencensus.NewHandlerWrapper()),
    micro.WrapSubscriber(opencensus.NewSubscriberWrapper()),
)
```

### Views

The OpenCensus package exposes some convenience views.
Don't forget to register these views:

```go
// Register to all RPC server views.
if err := view.Register(opencensus.DefaultServerViews...); err != nil {
    log.Fatal(err)
}

// Register to all RPC client views.
if err := view.Register(opencensus.DefaultClientViews...); err != nil {
    log.Fatal(err)
}
```
