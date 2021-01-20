package main

import (
	"fmt"

	"context"
	example "github.com/asim/go-micro/examples/v3/server/proto/example"
	"github.com/asim/go-micro/v3/cmd"
)

var (
	cl = example.NewExampleService("go.micro.srv.example", nil)
)

func call(i int) {
	rsp, err := cl.Call(context.Background(), &example.Request{Name: "John"})
	if err != nil {
		fmt.Println("call err: ", err, rsp)
		return
	}
	fmt.Println("Call:", i, "rsp:", rsp.Msg)
}

func stream(i int) {
	stream, err := cl.Stream(context.Background(), &example.StreamingRequest{Count: int64(i)})
	if err != nil {
		fmt.Println("err:", err)
		return
	}
	for j := 0; j < i; j++ {
		rsp, err := stream.Recv()
		if err != nil {
			fmt.Println("err:", err)
			break
		}
		fmt.Println("Stream: rsp:", rsp.Count)
	}
	if err := stream.Close(); err != nil {
		fmt.Println("stream close err:", err)
	}
}

func pingPong(i int) {
	stream, err := cl.PingPong(context.Background())
	if err != nil {
		fmt.Println("err:", err)
		return
	}
	for j := 0; j < i; j++ {
		if err := stream.Send(&example.Ping{Stroke: int64(j)}); err != nil {
			fmt.Println("err:", err)
			return
		}
		rsp, err := stream.Recv()
		if err != nil {
			fmt.Println("recv err", err)
			break
		}
		fmt.Printf("Sent ping %v got pong %v\n", j, rsp.Stroke)
	}
	if err := stream.Close(); err != nil {
		fmt.Println("stream close err:", err)
	}
}

func main() {
	cmd.Init()

	fmt.Println("\n--- Call example ---")
	for i := 0; i < 10; i++ {
		call(i)
	}

	fmt.Println("\n--- Streamer example ---")
	stream(10)

	fmt.Println("\n--- Ping Pong example ---")
	pingPong(10)
}
