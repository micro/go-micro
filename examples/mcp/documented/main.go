// Package main demonstrates how to document your service handlers for better
// AI agent integration using endpoint metadata.
//
// Services register descriptions with their endpoints, and the MCP gateway
// reads these descriptions from the registry to generate rich tool descriptions.
package main

import (
	"context"
	"fmt"
	"log"

	"go-micro.dev/v5"
	"go-micro.dev/v5/gateway/mcp"
	"go-micro.dev/v5/server"
)

// User represents a user in the system
type User struct {
	ID    string `json:"id" description:"User's unique identifier (UUID format)"`
	Name  string `json:"name" description:"User's full name"`
	Email string `json:"email" description:"User's email address"`
	Age   int    `json:"age,omitempty" description:"User's age (optional)"`
}

// GetUserRequest is the request for getting a user
type GetUserRequest struct {
	ID string `json:"id" description:"User ID to retrieve"`
}

// GetUserResponse is the response containing user data
type GetUserResponse struct {
	User *User `json:"user" description:"The requested user object"`
}

// CreateUserRequest is the request for creating a user
type CreateUserRequest struct {
	Name  string `json:"name" description:"User's full name (required)"`
	Email string `json:"email" description:"User's email address (required)"`
	Age   int    `json:"age,omitempty" description:"User's age (optional)"`
}

// CreateUserResponse contains the newly created user
type CreateUserResponse struct {
	User *User `json:"user" description:"The newly created user"`
}

// Users service handles user-related operations
type Users struct {
	users map[string]*User
}

// GetUser retrieves a user by ID from the database. Returns full profile including email, name, and preferences. If the user doesn't exist, an error is returned.
//
// @example {"id": "user-1"}
func (u *Users) GetUser(ctx context.Context, req *GetUserRequest, rsp *GetUserResponse) error {
	user, exists := u.users[req.ID]
	if !exists {
		return fmt.Errorf("user not found: %s", req.ID)
	}

	rsp.User = user
	return nil
}

// CreateUser creates a new user in the system. Validates the user data and creates a new profile. Name and email are required fields, while age is optional. Email must be unique across all users.
//
// @example {"name": "Alice Smith", "email": "alice@example.com", "age": 30}
func (u *Users) CreateUser(ctx context.Context, req *CreateUserRequest, rsp *CreateUserResponse) error {
	// Validate input
	if req.Name == "" || req.Email == "" {
		return fmt.Errorf("name and email are required")
	}

	// Generate ID (simplified for example)
	id := fmt.Sprintf("user-%d", len(u.users)+1)

	user := &User{
		ID:    id,
		Name:  req.Name,
		Email: req.Email,
		Age:   req.Age,
	}

	u.users[id] = user
	rsp.User = user

	return nil
}

func main() {
	// Create service
	service := micro.NewService(
		micro.Name("users"),
		micro.Version("1.0.0"),
	)

	service.Init()

	// Register handler with pre-populated test data
	usersService := &Users{
		users: map[string]*User{
			"user-1": {
				ID:    "user-1",
				Name:  "John Doe",
				Email: "john@example.com",
				Age:   25,
			},
			"user-2": {
				ID:    "user-2",
				Name:  "Jane Smith",
				Email: "jane@example.com",
				Age:   30,
			},
		},
	}

	// Register handler - documentation is automatically extracted from method comments.
	// Use WithEndpointScopes to declare required auth scopes per endpoint.
	handler := service.Server().NewHandler(
		usersService,
		server.WithEndpointScopes("Users.GetUser", "users:read"),
		server.WithEndpointScopes("Users.CreateUser", "users:write"),
	)

	if err := service.Server().Handle(handler); err != nil {
		log.Fatal(err)
	}

	// Start MCP gateway on port 3000
	go func() {
		log.Println("Starting MCP gateway on :3000")
		if err := mcp.ListenAndServe(":3000", mcp.Options{
			Registry: service.Options().Registry,
		}); err != nil {
			log.Printf("MCP gateway error: %v", err)
		}
	}()

	log.Println("Users service starting...")
	log.Println("Service: users")
	log.Println("Endpoints:")
	log.Println("  - Users.GetUser")
	log.Println("  - Users.CreateUser")
	log.Println("MCP Gateway: http://localhost:3000")
	log.Println("")
	log.Println("Test with:")
	log.Println("  curl http://localhost:3000/mcp/tools")
	log.Println("")
	log.Println("Or add to Claude Code:")
	log.Println(`  "users-service": {`)
	log.Println(`    "command": "micro",`)
	log.Println(`    "args": ["mcp", "serve"]`)
	log.Println(`  }`)

	// Run service
	if err := service.Run(); err != nil {
		log.Fatal(err)
	}
}
