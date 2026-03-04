// Multi-service example: run multiple services in a single binary.
//
// Each service gets its own server, client, store, and cache while
// sharing the registry, broker, and transport — so they can
// discover and call each other within the same process.
package main

import (
	"context"
	"fmt"
	"log"

	"go-micro.dev/v5/service"
)

// -- Users service --

type UserRequest struct {
	Id string `json:"id"`
}

type UserResponse struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Users struct{}

func (u *Users) Lookup(ctx context.Context, req *UserRequest, rsp *UserResponse) error {
	log.Printf("[users] Lookup id=%s", req.Id)
	rsp.Name = "Alice"
	rsp.Email = "alice@example.com"
	return nil
}

// -- Orders service --

type OrderRequest struct {
	UserId string `json:"user_id"`
}

type OrderResponse struct {
	OrderId string `json:"order_id"`
	Status  string `json:"status"`
}

type Orders struct{}

func (o *Orders) Create(ctx context.Context, req *OrderRequest, rsp *OrderResponse) error {
	log.Printf("[orders] Create for user=%s", req.UserId)
	rsp.OrderId = "ORD-001"
	rsp.Status = "created"
	return nil
}

func main() {
	// Create two services — each gets isolated server, client,
	// store, and cache instances automatically.
	users := service.New(
		service.Name("users"),
		service.Address(":9001"),
	)

	orders := service.New(
		service.Name("orders"),
		service.Address(":9002"),
	)

	// Register handlers
	if err := users.Handle(new(Users)); err != nil {
		log.Fatal(err)
	}
	if err := orders.Handle(new(Orders)); err != nil {
		log.Fatal(err)
	}

	// Run both services together. The group handles signals
	// and stops all services when one exits.
	g := service.NewGroup(users, orders)

	fmt.Println("Starting users (:9001) and orders (:9002) in a single binary")
	if err := g.Run(); err != nil {
		log.Fatal(err)
	}
}
