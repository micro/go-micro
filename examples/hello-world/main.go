package main

import (
	"context"
	"fmt"
	"log"

	"go-micro.dev/v5"
	"go-micro.dev/v5/client"
)

// Request and Response types
type Request struct {
	Name string `json:"name"`
}

type Response struct {
	Message string `json:"message"`
}

// Greeter service handler
type Greeter struct{}

// Hello is the RPC method handler
func (g *Greeter) Hello(ctx context.Context, req *Request, rsp *Response) error {
	rsp.Message = "Hello " + req.Name
	log.Printf("Received request: %s", req.Name)
	return nil
}

func main() {
	// Create a new service
	service := micro.New(
		micro.Name("greeter"),
		micro.Version("latest"),
		micro.Address(":8080"),
	)

	// Initialize the service
	service.Init()

	// Register the handler
	if err := service.Handle(new(Greeter)); err != nil {
		log.Fatal(err)
	}

	// Run the service in a goroutine
	go func() {
		if err := service.Run(); err != nil {
			log.Fatal(err)
		}
	}()

	// Wait for service to start
	fmt.Println("Service started on :8080")
	fmt.Println("Testing the service...")

	// Create a client to test the service
	c := service.Client()

	// Make a request
	req := c.NewRequest("greeter", "Greeter.Hello", &Request{Name: "World"})
	rsp := &Response{}

	if err := c.Call(context.Background(), req, rsp); err != nil {
		log.Printf("Error calling service: %v", err)
	} else {
		fmt.Printf("Response: %s\n", rsp.Message)
	}

	// Make another request
	req2 := c.NewRequest("greeter", "Greeter.Hello", &Request{Name: "Go Micro"})
	rsp2 := &Response{}

	if err := c.Call(context.Background(), req2, rsp2); err != nil {
		log.Printf("Error calling service: %v", err)
	} else {
		fmt.Printf("Response: %s\n", rsp2.Message)
	}

	// Test with HTTP client
	fmt.Println("\nYou can also test with curl:")
	fmt.Println("curl -X POST http://localhost:8080 \\")
	fmt.Println("  -H 'Content-Type: application/json' \\")
	fmt.Println("  -H 'Micro-Endpoint: Greeter.Hello' \\")
	fmt.Println("  -d '{\"name\": \"Alice\"}'")

	// Keep service running
	select {}
}
