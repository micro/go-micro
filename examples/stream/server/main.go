package main

import (
	"context"
	"fmt"
	"io"
	"log"

	proto "github.com/asim/go-micro/examples/v3/stream/server/proto"
	"github.com/asim/go-micro/v3"
)

type Streamer struct{}

// Server side stream
func (e *Streamer) ServerStream(ctx context.Context, req *proto.Request, stream proto.Streamer_ServerStreamStream) error {
	fmt.Printf("ServerStream Got msg %v\n", req.Count)
	for i := 0; i < int(req.Count); i++ {
		fmt.Println("sent", i)
		if err := stream.Send(&proto.Response{Count: int64(i)}); err != nil {
			return err
		}
	}
	return nil
}

// Bidirectional stream
func (e *Streamer) Stream(ctx context.Context, stream proto.Streamer_StreamStream) error {
	fmt.Println("Stream")
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		fmt.Printf("Got msg %v\n", req.Count)
		if err := stream.Send(&proto.Response{Count: req.Count}); err != nil {
			return err
		}
	}
}

func main() {
	// new service
	service := micro.NewService(
		micro.Name("go.micro.srv.stream"),
	)

	// Init command line
	service.Init()

	// Register Handler
	proto.RegisterStreamerHandler(service.Server(), new(Streamer))

	// Run service
	if err := service.Run(); err != nil {
		log.Fatal(err)
	}
}
