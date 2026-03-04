package main

import (
	"context"
	"fmt"
	"log"

	"go-micro.dev/v5"
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
	service := micro.New("greeter", micro.Address(":8080"))

	// Initialize the service
	service.Init()

	// Register the handler
	if err := service.Handle(new(Greeter)); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Starting greeter service on :8080")
	fmt.Println()
	fmt.Println("Test with:")
	fmt.Println("  curl -XPOST \\")
	fmt.Println("    -H 'Content-Type: application/json' \\")
	fmt.Println("    -H 'Micro-Endpoint: Greeter.Hello' \\")
	fmt.Println("    -d '{\"name\": \"Alice\"}' \\")
	fmt.Println("    http://localhost:8080")

	// Run the service
	if err := service.Run(); err != nil {
		log.Fatal(err)
	}
}
