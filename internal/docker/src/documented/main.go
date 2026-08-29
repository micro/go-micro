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
	"log/slog"
	"sync"

	"go-micro.dev/v6"
	natsbroker "go-micro.dev/v6/broker/nats"
	"go-micro.dev/v6/gateway/mcp"
	"go-micro.dev/v6/registry/nats"
	"go-micro.dev/v6/server"
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
	mu    sync.RWMutex
	users map[string]*User
}

// GetUser retrieves a user by ID from the database. Returns full profile including email, name, and preferences. If the user doesn't exist, an error is returned.
//
// @example {"id": "user-1"}
func (u *Users) GetUser(ctx context.Context, req *GetUserRequest, rsp *GetUserResponse) error {
	u.mu.RLock()
	defer u.mu.RUnlock()

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
	u.mu.Lock()
	defer u.mu.Unlock()

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
	rnats := nats.NewNatsRegistry()
	bnats := natsbroker.NewNatsBroker()
	service := micro.NewService(
		"users",
		micro.Address(":9090"),
		// Start MCP gateway alongside the service
		mcp.WithMCP(":3000"),
		micro.Registry(rnats),
		micro.Broker(bnats),
	)

	service.Init()

	// Register handler with pre-populated test data.
	// Documentation is automatically extracted from method comments.
	// Use WithEndpointScopes to declare required auth scopes per endpoint.
	if err := service.Handle(
		&Users{
			users: map[string]*User{
				"user-1": {ID: "user-1", Name: "John Doe", Email: "john@example.com", Age: 25},
				"user-2": {ID: "user-2", Name: "Jane Smith", Email: "jane@example.com", Age: 30},
			},
		},
		server.WithEndpointScopes("Users.GetUser", "users:read"),
		server.WithEndpointScopes("Users.CreateUser", "users:write"),
	); err != nil {
		log.Fatal(err)
	}

	slog.Info("started", "service", "users", "mcp", "http://localhost:3000/mcp/tools")

	// Run service
	if err := service.Run(); err != nil {
		log.Fatal(err)
	}
}
