package main

import (
	"context"
	"fmt"

	proto "github.com/asim/go-micro/examples/v4/service/proto"
	"go-micro.dev/v4"
	"go-micro.dev/v4/client"
)

// log wrapper logs every time a request is made
type logWrapper struct {
	client.Client
}

func (l *logWrapper) Call(ctx context.Context, req client.Request, rsp interface{}, opts ...client.CallOption) error {
	fmt.Printf("[wrapper] client request service: %s method: %s\n", req.Service(), req.Endpoint())
	return l.Client.Call(ctx, req, rsp, opts...)
}

func (l *logWrapper) Publish(ctx context.Context, msg client.Message, opts ...client.PublishOption) error {
	fmt.Printf("[wrapper] client publish: %#v\n", msg)
	return l.Client.Publish(ctx, msg, opts...)
}

// Implements client.Wrapper as logWrapper
func logWrap(c client.Client) client.Client {
	return &logWrapper{c}
}

func logPublish(fn client.PublishFunc) client.PublishFunc {
	return func(ctx context.Context, msg client.Message, opts ...client.PublishOption) error {
		fmt.Printf("[logPublish] %#v\n", msg)
		return fn(ctx, msg, opts...)
	}
}

func main() {
	service := micro.NewService(
		micro.Name("greeter.client"),
		// wrap the client
		micro.WrapClient(logWrap),
		micro.WrapPublish(logPublish),
	)

	service.Init()

	greeter := proto.NewGreeterService("greeter", service.Client())

	rsp, err := greeter.Hello(context.TODO(), &proto.Request{Name: "John"})
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(rsp.Greeting)

	if err := service.Client().Publish(context.TODO(), client.NewMessage("greeter", &proto.Request{Name: "John"})); err != nil {
		fmt.Println(err)
	}
}
