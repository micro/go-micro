// client calls the go-micro helloworld service using a standard gRPC client
// — no go-micro SDK. This proves any gRPC client (grpcurl, Python, Java, ...)
// can call a go-micro service.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "example/proto"
)

func main() {
	addr := flag.String("addr", "localhost:8080", "server address")
	name := flag.String("name", "World", "name to greet")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.NewClient(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer conn.Close()

	resp, err := pb.NewSayClient(conn).Hello(ctx, &pb.Request{Name: *name})
	if err != nil {
		log.Fatalf("Say.Hello: %v", err)
	}
	fmt.Printf("Response: %s\n", resp.Message)
}
