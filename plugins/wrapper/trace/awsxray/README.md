# AWS X-Ray Wrappers

AWS X-Ray wrappers make use of either the AWS API or the X-Ray Daemon.

## Usage

```go
opts := []awsxray.Option{
	// Used as segment name
	awsxray.WithName("go.micro.srv.greeter"),
	// Specify X-Ray Daemon Address
	awsxray.WithDaemon("localhost:2000"),
	// Or X-Ray Client
	awsxray.WithClient(xray.New(awsSession)),
}

service := micro.NewService(
	micro.Name("go.micro.srv.greeter"),
	micro.WrapCall(awsxray.NewCallWrapper(opts...)),
	micro.WrapClient(awsxray.NewClientWrapper(opts...)),
	micro.WrapHandler(awsxray.NewHandlerWrapper(opts...)),
)
```

## Example

<p align="center">
  <img src="awsxray.png" />
</p>
