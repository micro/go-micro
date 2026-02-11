package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"go-micro.dev/v5"
	"go-micro.dev/v5/auth"
	"go-micro.dev/v5/auth/jwt"
	"go-micro.dev/v5/auth/noop"
	"go-micro.dev/v5/client"
	authWrapper "go-micro.dev/v5/wrapper/auth"

	pb "go-micro.dev/v5/examples/auth/proto"
)

func main() {
	// Get token from environment or generate one
	token := os.Getenv("TOKEN")

	// Create auth provider (same as server)
	var authProvider auth.Auth
	var err error

	authProvider, err = jwt.NewAuth(
		auth.Issuer("go-micro"),
		auth.Store(nil),
	)
	if err != nil {
		log.Printf("JWT auth failed, falling back to noop: %v", err)
		authProvider = noop.NewAuth()
	}

	// If no token provided, generate one
	if token == "" {
		log.Println("No TOKEN env var provided, generating test token...")
		acc, err := authProvider.Generate("test-user")
		if err != nil {
			log.Fatal(err)
		}

		t, err := authProvider.Token(auth.WithCredentials(acc.ID, acc.Secret))
		if err != nil {
			log.Fatal(err)
		}

		token = t.AccessToken
		log.Printf("Generated token: %s\n", token)
	}

	// Create service with auth client wrapper
	service := micro.NewService(
		micro.Name("greeter.client"),
		micro.WrapClient(
			authWrapper.FromToken(token), // Add token to all requests
		),
	)

	service.Init()

	// Create greeter client
	greeterClient := pb.NewGreeterService("greeter", service.Client())

	// Test 1: Call protected endpoint (Hello) with auth
	fmt.Println("\n=== Test 1: Protected endpoint WITH auth ===")
	rsp, err := greeterClient.Hello(context.Background(), &pb.Request{Name: "John"})
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("Response: %s\n", rsp.Msg)
	}

	// Test 2: Call public endpoint (Health) without auth
	fmt.Println("\n=== Test 2: Public endpoint (no auth needed) ===")
	// Create client without auth wrapper for this test
	plainClient := client.NewClient()
	plainGreeterClient := pb.NewGreeterService("greeter", plainClient)

	healthRsp, err := plainGreeterClient.Health(context.Background(), &pb.HealthRequest{})
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("Health Status: %s\n", healthRsp.Status)
	}

	// Test 3: Call protected endpoint WITHOUT auth (should fail)
	fmt.Println("\n=== Test 3: Protected endpoint WITHOUT auth (should fail) ===")
	_, err = plainGreeterClient.Hello(context.Background(), &pb.Request{Name: "John"})
	if err != nil {
		fmt.Printf("Expected error: %v\n", err)
	} else {
		fmt.Println("Unexpected: Call succeeded without auth!")
	}
}
