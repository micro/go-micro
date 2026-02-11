package main

import (
	"context"
	"log"

	"go-micro.dev/v5"
	"go-micro.dev/v5/auth"
	"go-micro.dev/v5/auth/noop"
	authWrapper "go-micro.dev/v5/wrapper/auth"

	pb "go-micro.dev/v5/examples/auth/proto"
)

// Greeter implements the Greeter service
type Greeter struct{}

// Hello is a protected endpoint that requires authentication
func (g *Greeter) Hello(ctx context.Context, req *pb.Request, rsp *pb.Response) error {
	// Get account from context (added by auth wrapper)
	acc, ok := auth.AccountFromContext(ctx)
	if !ok {
		rsp.Msg = "Hello, anonymous!"
		return nil
	}

	rsp.Msg = "Hello, " + acc.ID + "!"
	return nil
}

// Health is a public endpoint that doesn't require auth
func (g *Greeter) Health(ctx context.Context, req *pb.HealthRequest, rsp *pb.HealthResponse) error {
	rsp.Status = "ok"
	return nil
}

func main() {
	// Create auth provider (noop for this example)
	// In production, use JWT or custom auth provider
	authProvider := noop.NewAuth()

	// Create authorization rules
	rules := auth.NewRules()

	// Grant public access to health endpoint
	rules.Grant(&auth.Rule{
		ID:       "public-health",
		Scope:    "",
		Resource: &auth.Resource{Type: "service", Name: "*", Endpoint: "Greeter.Health"},
		Access:   auth.AccessGranted,
		Priority: 100,
	})

	// Require authentication for other endpoints
	rules.Grant(&auth.Rule{
		ID:       "authenticated-hello",
		Scope:    "*",
		Resource: &auth.Resource{Type: "service", Name: "*", Endpoint: "*"},
		Access:   auth.AccessGranted,
		Priority: 50,
	})

	// Create service with auth wrapper
	service := micro.NewService(
		micro.Name("greeter"),
		micro.Version("latest"),
		micro.WrapHandler(
			authWrapper.AuthHandler(authWrapper.HandlerOptions{
				Auth:          authProvider,
				Rules:         rules,
				SkipEndpoints: []string{"Greeter.Health"}, // Public endpoints
			}),
		),
	)

	service.Init()

	// Register handler
	if err := pb.RegisterGreeterHandler(service.Server(), &Greeter{}); err != nil {
		log.Fatal(err)
	}

	// Generate a test token for demonstration
	if acc, err := authProvider.Generate("test-user"); err == nil {
		if token, err := authProvider.Token(auth.WithCredentials(acc.ID, acc.Secret)); err == nil {
			log.Printf("\n=== Test Token Generated ===")
			log.Printf("Use this token to test the client:")
			log.Printf("TOKEN=%s go run client/main.go\n", token.AccessToken)
		}
	}

	// Run service
	if err := service.Run(); err != nil {
		log.Fatal(err)
	}
}
