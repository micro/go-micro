// client calls the go-micro Greeter service using a standard gRPC
// client — no go-micro SDK. This proves any language with gRPC support
// can call go-micro services.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	pb "example/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	addr := flag.String("addr", "localhost:50051", "server address")
	name := flag.String("name", "World", "name to greet")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.NewClient(*addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewGreeterClient(conn)

	resp, err := client.Hello(ctx, &pb.HelloRequest{Name: *name})
	if err != nil {
		log.Fatalf("Hello: %v", err)
	}

	fmt.Printf("Response: %s\n", resp.Message)
}
