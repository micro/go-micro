package main

import (
	"fmt"

	"context"
	example "github.com/asim/go-micro/examples/v3/server/proto/example"
	"github.com/asim/go-micro/v3"
	"github.com/asim/go-micro/v3/client"
	"github.com/asim/go-micro/v3/metadata"
)

// publishes a message
func pub(p micro.Publisher) {
	msg := &example.Message{
		Say: "This is an async message",
	}

	if err := p.Publish(context.TODO(), msg); err != nil {
		fmt.Println("pub err: ", err)
		return
	}

	fmt.Printf("Published: %v\n", msg)
}

func call(i int, c client.Client) {
	// Create new request to service go.micro.srv.example, method Example.Call
	req := c.NewRequest("go.micro.srv.example", "Example.Call", &example.Request{
		Name: "John",
	})

	// create context with metadata
	ctx := metadata.NewContext(context.Background(), map[string]string{
		"X-User-Id": "john",
		"X-From-Id": "script",
	})

	rsp := &example.Response{}

	// Call service
	if err := c.Call(ctx, req, rsp); err != nil {
		fmt.Println("call err: ", err, rsp)
		return
	}

	fmt.Println("Call:", i, "rsp:", rsp.Msg)
}

func stream(i int, c client.Client) {
	// Create new request to service go.micro.srv.example, method Example.Call
	// Request can be empty as its actually ignored and merely used to call the handler
	req := c.NewRequest("go.micro.srv.example", "Example.Stream", &example.StreamingRequest{})

	stream, err := c.Stream(context.Background(), req)
	if err != nil {
		fmt.Println("err:", err)
		return
	}
	if err := stream.Send(&example.StreamingRequest{Count: int64(i)}); err != nil {
		fmt.Println("err:", err)
		return
	}
	for stream.Error() == nil {
		rsp := &example.StreamingResponse{}
		err := stream.Recv(rsp)
		if err != nil {
			fmt.Println("recv err", err)
			break
		}
		fmt.Println("Stream: rsp:", rsp.Count)
	}

	if stream.Error() != nil {
		fmt.Println("stream err:", err)
		return
	}

	if err := stream.Close(); err != nil {
		fmt.Println("stream close err:", err)
	}
}

func pingPong(i int, c client.Client) {
	// Create new request to service go.micro.srv.example, method Example.Call
	// Request can be empty as its actually ignored and merely used to call the handler
	req := c.NewRequest("go.micro.srv.example", "Example.PingPong", &example.StreamingRequest{})

	stream, err := c.Stream(context.Background(), req)
	if err != nil {
		fmt.Println("err:", err)
		return
	}

	for j := 0; j < i; j++ {
		if err := stream.Send(&example.Ping{Stroke: int64(j + 1)}); err != nil {
			fmt.Println("err:", err)
			return
		}
		rsp := &example.Pong{}
		err := stream.Recv(rsp)
		if err != nil {
			fmt.Println("recv err", err)
			break
		}
		fmt.Printf("Sent ping %v got pong %v\n", j+1, rsp.Stroke)
	}

	if stream.Error() != nil {
		fmt.Println("stream err:", err)
		return
	}

	if err := stream.Close(); err != nil {
		fmt.Println("stream close err:", err)
	}
}

func main() {
	service := micro.NewService()
	service.Init()

	p := micro.NewPublisher("topic.example", service.Client())

	fmt.Println("\n--- Publisher example ---")
	pub(p)

	fmt.Println("\n--- Call example ---")
	for i := 0; i < 10; i++ {
		call(i, service.Client())
	}

	fmt.Println("\n--- Streamer example ---")
	stream(10, service.Client())

	fmt.Println("\n--- Ping Pong example ---")
	pingPong(10, service.Client())

}
