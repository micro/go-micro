# Service Wrapper

A lot of the time you need access to the service from within the handler. The service wrapper provides a way 
to do that. It embeds the service within the context so that it can then be retrieved within the handler.

## Usage

When creating your service, add the wrapper like so.

```go
srv := micro.NewService(
	micro.Name("com.example.srv.foo"),
)

srv.Init(
	micro.WrapClient(service.NewClientWrapper(srv))
	micro.WrapHandler(service.NewHandlerWrapper(srv)),
)

```

Then within your handler it can be accessed like this.

```go
func (e *Example) Handler(ctx context.Context, req *example.Request, rsp *example.Response) error {
	service, ok := micro.FromContext(ctx)
	if !ok {
		return errors.InternalServerError("com.example.srv.foo", "Could not retrieve service")
	}

	// do something with the service
	fmt.Println("Got service", service)
	return nil
}
```

And if you decide to wrap the client with some other wrapper it becomes accessible there too.

```go
type myWrapper struct {
	client.Client
}

func (m *myWrapper) Call(ctx context.Context, req client.Request, rsp interface{}, opts ...client.CallOption) error {
	service, ok = micro.FromContext(ctx)
	if !ok {
		return errors.InternalServerError("com.example.srv.foo.mywrapper", "Could not retrieve service")
	}

	// do something with the service
	fmt.Println("Got service", service)

	// now do some call
	return c.Client.Call(ctx, req, rsp, opts...)
}
```
