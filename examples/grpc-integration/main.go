// gRPC Integration example: using go-micro with gRPC transport.
//
// This example demonstrates:
//   - gRPC server with reflection-based handler registration
//   - gRPC client with retries and timeouts
//   - JSON codec (no protobuf compilation needed)
//   - Streaming RPC
//
// The gRPC transport is a drop-in replacement — same handler code,
// different wire protocol.
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"go-micro.dev/v5"
	"go-micro.dev/v5/client"
	grpccli "go-micro.dev/v5/client/grpc"
	"go-micro.dev/v5/codec/bytes"
	grpcsrv "go-micro.dev/v5/server/grpc"
)

// -- Request/Response types --

type EchoRequest struct {
	Message string `json:"message"`
}

type EchoResponse struct {
	Message string `json:"message"`
	Server  string `json:"server"`
}

type StreamRequest struct {
	Count int `json:"count"`
}

type StreamResponse struct {
	Seq     int    `json:"seq"`
	Message string `json:"message"`
}

// -- Handler --

type Echo struct{}

// Call is a unary RPC — same signature as standard go-micro handlers.
// Works with both gRPC and default RPC transports.
func (e *Echo) Call(ctx context.Context, req *EchoRequest, rsp *EchoResponse) error {
	log.Printf("[echo] Received: %s", req.Message)
	rsp.Message = "echo: " + req.Message
	rsp.Server = "grpc"
	return nil
}

// Reverse echoes the message in reverse
func (e *Echo) Reverse(ctx context.Context, req *EchoRequest, rsp *EchoResponse) error {
	runes := []rune(req.Message)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	rsp.Message = string(runes)
	rsp.Server = "grpc"
	return nil
}

func main() {
	// Create a service with gRPC server and client.
	// The handler code is identical — only the transport changes.
	svc := micro.New("echo",
		micro.Address(":9004"),
		micro.Server(grpcsrv.NewServer()),
		micro.Client(grpccli.NewClient()),
	)

	svc.Init()

	if err := svc.Handle(new(Echo)); err != nil {
		log.Fatal(err)
	}

	// Start the server in background so we can demo the client
	go func() {
		if err := svc.Run(); err != nil {
			log.Fatal(err)
		}
	}()

	// Give server time to start
	time.Sleep(500 * time.Millisecond)

	// -- Client demo --
	fmt.Println("=== gRPC Client Demo ===")
	fmt.Println()

	cli := grpccli.NewClient()

	// Unary call with retries
	req := cli.NewRequest("echo", "Echo.Call", &EchoRequest{
		Message: "hello from grpc client",
	}, client.WithContentType("application/json"))

	var rsp EchoResponse
	if err := cli.Call(context.Background(), req, &rsp, client.WithRetries(3)); err != nil {
		log.Fatalf("Call failed: %v", err)
	}
	fmt.Printf("  Echo.Call response: %s (server: %s)\n", rsp.Message, rsp.Server)

	// Call another method
	req2 := cli.NewRequest("echo", "Echo.Reverse", &EchoRequest{
		Message: "grpc works!",
	}, client.WithContentType("application/json"))

	var rsp2 EchoResponse
	if err := cli.Call(context.Background(), req2, &rsp2); err != nil {
		log.Fatalf("Call failed: %v", err)
	}
	fmt.Printf("  Echo.Reverse response: %s\n", rsp2.Message)

	// Raw bytes call (useful for proxying or dynamic payloads)
	rawReq := cli.NewRequest("echo", "Echo.Call", &bytes.Frame{
		Data: []byte(`{"message": "raw bytes call"}`),
	})
	var rawRsp bytes.Frame
	if err := cli.Call(context.Background(), rawReq, &rawRsp); err != nil {
		log.Fatalf("Raw call failed: %v", err)
	}
	fmt.Printf("  Raw bytes response: %s\n", string(rawRsp.Data))

	fmt.Println()
	fmt.Println("=== Service Running ===")
	fmt.Println()
	fmt.Println("Test with curl:")
	fmt.Println("  curl -XPOST \\")
	fmt.Println("    -H 'Content-Type: application/json' \\")
	fmt.Println("    -H 'Micro-Endpoint: Echo.Call' \\")
	fmt.Println("    -d '{\"message\": \"hi\"}' \\")
	fmt.Println("    http://localhost:9004")
	fmt.Println()
	fmt.Println("Or with micro CLI:")
	fmt.Println("  micro call echo Echo.Call '{\"message\": \"hi\"}'")
	fmt.Println("  micro call echo Echo.Reverse '{\"message\": \"hello\"}'")
	fmt.Println()
	fmt.Println("Press Ctrl+C to stop")

	// Block forever (server is already running in goroutine)
	select {}
}
