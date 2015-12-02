package main

import (
	"fmt"

	"github.com/micro/go-micro/cmd"
	example "github.com/micro/go-micro/examples/server/proto/example"
	"golang.org/x/net/context"
)

var (
	cl = example.NewExampleClient(nil)
)

func call(i int) {
	rsp, err := cl.Call(context.Background(), &example.Request{Name: "John"})
	if err != nil {
		fmt.Println("call err: ", err, rsp)
		return
	}
	fmt.Println("Call:", i, "rsp:", rsp.Msg)
}

func stream() {
	stream, err := cl.Stream(context.Background(), &example.StreamingRequest{Count: int64(10)})
	if err != nil {
		fmt.Println("err:", err)
		return
	}
	for i := 0; i < 10; i++ {
		rsp, err := stream.Next()
		if err != nil {
			fmt.Println("err:", err)
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

func main() {
	cmd.Init()

	fmt.Println("\n--- Call example ---\n")
	for i := 0; i < 10; i++ {
		call(i)
	}
	fmt.Println("\n--- Streamer example ---\n")
	stream()
}
