package main

import (
	"fmt"

	"context"
	proto "github.com/asim/go-micro/examples/v3/service/proto"
	"github.com/asim/go-micro/v3"
	"github.com/asim/go-micro/v3/client"
)

// log wrapper logs every time a request is made
type logWrapper struct {
	client.Client
}

func (l *logWrapper) Call(ctx context.Context, req client.Request, rsp interface{}, opts ...client.CallOption) error {
	fmt.Printf("[wrapper] client request service: %s method: %s\n", req.Service(), req.Endpoint())
	return l.Client.Call(ctx, req, rsp)
}

// Implements client.Wrapper as logWrapper
func logWrap(c client.Client) client.Client {
	return &logWrapper{c}
}

func main() {
	service := micro.NewService(
		micro.Name("greeter.client"),
		// wrap the client
		micro.WrapClient(logWrap),
	)

	service.Init()

	greeter := proto.NewGreeterService("greeter", service.Client())

	rsp, err := greeter.Hello(context.TODO(), &proto.Request{Name: "John"})
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(rsp.Greeting)

}
