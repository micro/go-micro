# Endpoint Wrapper

The endpoint wrapper is a function which allows you to execute a wrapper at a more granular level. 
At the moment client or handler wrappers are executed on any request method. The endpoint wrapper 
makes it much easier to specify exact methods to execute on otherwise acting as a pass through.

## Usage

When creating your service, add the wrapper like so.

```go
srv := micro.NewService(
	micro.Name("com.example.srv.foo"),
)

srv.Init(
	// cw is your client wrapper
	// hw is your handler wrapper
	// Foo.Bar and Foo.Baz are the methods to execute on
	micro.WrapClient(endpoint.NewClientWrapper(cw, "Foo.Bar", "Foo.Baz"))
	micro.WrapHandler(endpoint.NewHandlerWrapper(hw, "Foo.Bar", "Bar.Baz", "Debug.Health")),
)

```
