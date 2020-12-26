# Mocking

Thie example demonstrates how to mock the helloworld service

The generated protos create a `Service` interface used by the client. This can simply be mocked.

```go
type GreeterService interface {
	Hello(ctx context.Context, in *Request, opts ...client.CallOption) (*Response, error)
}
```

Where the `GreeterService` is used we can instead pass in the mock which returns the expected response rather than calling a service.

## Mock Client

```go
type mockGreeterService struct {
}

func (m *mockGreeterService) Hello(ctx context.Context, req *proto.Request, opts ...client.CallOption) (*proto.Response, error) {
        return &proto.Response{
                Greeting: "Hello " + req.Name,
        }, nil
}

func NewGreeterService() proto.GreeterService {
        return new(mockGreeterService)
}
```

## Use Mock

In the test environment we will use the mock client

```go
func main() {
	var c proto.GreeterService

	service := micro.NewService(
		micro.Flags(&cli.StringFlag{
			Name: "environment",
			Value: "testing",
		}),
	)

	service.Init(
		micro.Action(func(ctx *cli.Context) error {
			env := ctx.String("environment")
			// use the mock when in testing environment
			if env == "testing" {
				c = mock.NewGreeterService()
			} else {
				c = proto.NewGreeterService("helloworld", service.Client())
			}
                        return nil
		}),
	)

	// call hello service
	rsp, err := c.Hello(context.TODO(), &proto.Request{
		Name: "John",
	})
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(rsp.Greeting)
}
```
