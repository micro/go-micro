// Workflow example: cross-service orchestration via AI agents.
//
// This example runs three services (Inventory, Orders, Notifications) and
// demonstrates how an AI agent can orchestrate a multi-step workflow:
//
//	1. Check inventory for a product
//	2. Place an order if in stock
//	3. Send a confirmation notification
//
// The agent figures out the right sequence of calls on its own — no
// workflow engine, no glue code, just natural language.
//
// Run:
//
//	go run .
//
// MCP tools: http://localhost:3001/mcp/tools
package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"go-micro.dev/v5"
	"go-micro.dev/v5/gateway/mcp"
)

// ---------------------------------------------------------------------------
// Inventory service
// ---------------------------------------------------------------------------

type Product struct {
	SKU      string  `json:"sku" description:"Stock keeping unit identifier"`
	Name     string  `json:"name" description:"Product name"`
	Price    float64 `json:"price" description:"Unit price in USD"`
	InStock  int     `json:"in_stock" description:"Number of units available"`
	Category string  `json:"category" description:"Product category"`
}

type CheckStockRequest struct {
	SKU string `json:"sku" description:"Product SKU to check"`
}

type CheckStockResponse struct {
	Product *Product `json:"product" description:"Product details with current stock level"`
}

type SearchProductsRequest struct {
	Query    string `json:"query" description:"Search term to match against product name or category"`
	Category string `json:"category,omitempty" description:"Filter by category: electronics, clothing, books (optional)"`
}

type SearchProductsResponse struct {
	Products []*Product `json:"products" description:"Products matching the search criteria"`
}

type ReserveStockRequest struct {
	SKU      string `json:"sku" description:"Product SKU to reserve"`
	Quantity int    `json:"quantity" description:"Number of units to reserve"`
}

type ReserveStockResponse struct {
	Reserved  bool   `json:"reserved" description:"True if stock was successfully reserved"`
	Remaining int    `json:"remaining" description:"Units remaining after reservation"`
	Message   string `json:"message" description:"Human-readable result message"`
}

type InventoryService struct {
	mu       sync.RWMutex
	products map[string]*Product
}

// CheckStock returns the current stock level for a product.
// Use this before placing an order to verify availability.
//
// @example {"sku": "LAPTOP-001"}
func (s *InventoryService) CheckStock(ctx context.Context, req *CheckStockRequest, rsp *CheckStockResponse) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.products[req.SKU]
	if !ok {
		return fmt.Errorf("product %s not found", req.SKU)
	}
	rsp.Product = p
	return nil
}

// Search finds products by name or category. Use this to help
// users find what they're looking for before checking stock.
//
// @example {"query": "laptop"}
func (s *InventoryService) Search(ctx context.Context, req *SearchProductsRequest, rsp *SearchProductsResponse) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	q := strings.ToLower(req.Query)
	for _, p := range s.products {
		if req.Category != "" && !strings.EqualFold(p.Category, req.Category) {
			continue
		}
		if q == "" || strings.Contains(strings.ToLower(p.Name), q) || strings.Contains(strings.ToLower(p.Category), q) {
			rsp.Products = append(rsp.Products, p)
		}
	}
	return nil
}

// ReserveStock decrements inventory for a product. Call this after
// confirming stock is available. Returns an error if insufficient stock.
//
// @example {"sku": "LAPTOP-001", "quantity": 1}
func (s *InventoryService) ReserveStock(ctx context.Context, req *ReserveStockRequest, rsp *ReserveStockResponse) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.products[req.SKU]
	if !ok {
		return fmt.Errorf("product %s not found", req.SKU)
	}
	if p.InStock < req.Quantity {
		rsp.Reserved = false
		rsp.Remaining = p.InStock
		rsp.Message = fmt.Sprintf("insufficient stock: requested %d but only %d available", req.Quantity, p.InStock)
		return nil
	}
	p.InStock -= req.Quantity
	rsp.Reserved = true
	rsp.Remaining = p.InStock
	rsp.Message = fmt.Sprintf("reserved %d units of %s", req.Quantity, p.Name)
	return nil
}

// ---------------------------------------------------------------------------
// Orders service
// ---------------------------------------------------------------------------

type Order struct {
	ID        string    `json:"id" description:"Unique order identifier"`
	Customer  string    `json:"customer" description:"Customer name or email"`
	SKU       string    `json:"sku" description:"Product SKU ordered"`
	Quantity  int       `json:"quantity" description:"Number of units"`
	Total     float64   `json:"total" description:"Total order amount in USD"`
	Status    string    `json:"status" description:"Order status: pending, confirmed, shipped, delivered"`
	CreatedAt time.Time `json:"created_at" description:"When the order was placed"`
}

type PlaceOrderRequest struct {
	Customer string `json:"customer" description:"Customer name or email (required)"`
	SKU      string `json:"sku" description:"Product SKU to order (required)"`
	Quantity int    `json:"quantity" description:"Number of units (required, must be positive)"`
}

type PlaceOrderResponse struct {
	Order *Order `json:"order" description:"The newly created order"`
}

type GetOrderRequest struct {
	ID string `json:"id" description:"Order ID to look up"`
}

type GetOrderResponse struct {
	Order *Order `json:"order" description:"The requested order"`
}

type ListOrdersRequest struct {
	Customer string `json:"customer,omitempty" description:"Filter by customer (optional)"`
	Status   string `json:"status,omitempty" description:"Filter by status (optional)"`
}

type ListOrdersResponse struct {
	Orders []*Order `json:"orders" description:"Matching orders"`
}

type OrderService struct {
	mu     sync.RWMutex
	orders map[string]*Order
	nextID int
	// In a real app this would be a client to the inventory service
	inventory *InventoryService
}

// PlaceOrder creates a new order. Stock must be reserved first via
// InventoryService.ReserveStock — this service does not check inventory.
//
// @example {"customer": "alice@example.com", "sku": "LAPTOP-001", "quantity": 1}
func (s *OrderService) PlaceOrder(ctx context.Context, req *PlaceOrderRequest, rsp *PlaceOrderResponse) error {
	if req.Customer == "" {
		return fmt.Errorf("customer is required")
	}
	if req.SKU == "" {
		return fmt.Errorf("sku is required")
	}
	if req.Quantity <= 0 {
		return fmt.Errorf("quantity must be positive")
	}

	// Look up price
	s.inventory.mu.RLock()
	p, ok := s.inventory.products[req.SKU]
	s.inventory.mu.RUnlock()
	if !ok {
		return fmt.Errorf("product %s not found", req.SKU)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	order := &Order{
		ID:        fmt.Sprintf("ORD-%04d", s.nextID),
		Customer:  req.Customer,
		SKU:       req.SKU,
		Quantity:  req.Quantity,
		Total:     p.Price * float64(req.Quantity),
		Status:    "confirmed",
		CreatedAt: time.Now(),
	}
	s.orders[order.ID] = order
	rsp.Order = order
	return nil
}

// GetOrder retrieves an order by ID.
//
// @example {"id": "ORD-0001"}
func (s *OrderService) GetOrder(ctx context.Context, req *GetOrderRequest, rsp *GetOrderResponse) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	o, ok := s.orders[req.ID]
	if !ok {
		return fmt.Errorf("order %s not found", req.ID)
	}
	rsp.Order = o
	return nil
}

// ListOrders returns orders, optionally filtered by customer or status.
//
// @example {"customer": "alice@example.com"}
func (s *OrderService) ListOrders(ctx context.Context, req *ListOrdersRequest, rsp *ListOrdersResponse) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, o := range s.orders {
		if req.Customer != "" && o.Customer != req.Customer {
			continue
		}
		if req.Status != "" && o.Status != req.Status {
			continue
		}
		rsp.Orders = append(rsp.Orders, o)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Notifications service
// ---------------------------------------------------------------------------

type Notification struct {
	ID        string    `json:"id" description:"Notification identifier"`
	Recipient string    `json:"recipient" description:"Who received the notification"`
	Subject   string    `json:"subject" description:"Notification subject line"`
	Body      string    `json:"body" description:"Notification body text"`
	Channel   string    `json:"channel" description:"Delivery channel: email, sms, or slack"`
	SentAt    time.Time `json:"sent_at" description:"When the notification was sent"`
}

type SendNotificationRequest struct {
	Recipient string `json:"recipient" description:"Email address, phone number, or Slack handle"`
	Subject   string `json:"subject" description:"Subject line (required)"`
	Body      string `json:"body" description:"Message body (required)"`
	Channel   string `json:"channel,omitempty" description:"Channel: email (default), sms, or slack"`
}

type SendNotificationResponse struct {
	Notification *Notification `json:"notification" description:"The sent notification with delivery details"`
}

type ListNotificationsRequest struct {
	Recipient string `json:"recipient,omitempty" description:"Filter by recipient (optional)"`
}

type ListNotificationsResponse struct {
	Notifications []*Notification `json:"notifications" description:"Sent notifications"`
}

type NotificationService struct {
	mu            sync.RWMutex
	notifications []*Notification
	nextID        int
}

// Send delivers a notification to a recipient via the specified channel.
// Use this to confirm orders, alert users, or send updates.
// Defaults to email if no channel is specified.
//
// @example {"recipient": "alice@example.com", "subject": "Order Confirmed", "body": "Your order ORD-0001 has been confirmed.", "channel": "email"}
func (s *NotificationService) Send(ctx context.Context, req *SendNotificationRequest, rsp *SendNotificationResponse) error {
	if req.Recipient == "" {
		return fmt.Errorf("recipient is required")
	}
	if req.Subject == "" {
		return fmt.Errorf("subject is required")
	}
	if req.Body == "" {
		return fmt.Errorf("body is required")
	}
	channel := req.Channel
	if channel == "" {
		channel = "email"
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	n := &Notification{
		ID:        fmt.Sprintf("notif-%d", s.nextID),
		Recipient: req.Recipient,
		Subject:   req.Subject,
		Body:      req.Body,
		Channel:   channel,
		SentAt:    time.Now(),
	}
	s.notifications = append(s.notifications, n)
	rsp.Notification = n
	return nil
}

// List returns sent notifications, optionally filtered by recipient.
//
// @example {"recipient": "alice@example.com"}
func (s *NotificationService) List(ctx context.Context, req *ListNotificationsRequest, rsp *ListNotificationsResponse) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, n := range s.notifications {
		if req.Recipient != "" && n.Recipient != req.Recipient {
			continue
		}
		rsp.Notifications = append(rsp.Notifications, n)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

func main() {
	service := micro.New("shop",
		micro.Address(":9090"),
		mcp.WithMCP(":3001"),
	)
	service.Init()

	inventory := &InventoryService{products: map[string]*Product{
		"LAPTOP-001": {SKU: "LAPTOP-001", Name: "ThinkPad X1 Carbon", Price: 1299.99, InStock: 15, Category: "electronics"},
		"LAPTOP-002": {SKU: "LAPTOP-002", Name: "MacBook Air M3", Price: 1099.00, InStock: 8, Category: "electronics"},
		"PHONE-001":  {SKU: "PHONE-001", Name: "Pixel 8 Pro", Price: 899.00, InStock: 23, Category: "electronics"},
		"BOOK-001":   {SKU: "BOOK-001", Name: "Designing Data-Intensive Applications", Price: 45.99, InStock: 50, Category: "books"},
		"BOOK-002":   {SKU: "BOOK-002", Name: "The Go Programming Language", Price: 39.99, InStock: 0, Category: "books"},
		"SHIRT-001":  {SKU: "SHIRT-001", Name: "Go Gopher T-Shirt", Price: 24.99, InStock: 100, Category: "clothing"},
	}}

	orders := &OrderService{
		orders:    make(map[string]*Order),
		inventory: inventory,
	}

	notifications := &NotificationService{}

	service.Handle(inventory)
	service.Handle(orders)
	service.Handle(notifications)

	fmt.Println()
	fmt.Println("  Shop Workflow Demo")
	fmt.Println()
	fmt.Println("  MCP Tools:  http://localhost:3001/mcp/tools")
	fmt.Println()
	fmt.Println("  Try asking an agent:")
	fmt.Println()
	fmt.Println("    \"What laptops do you have in stock?\"")
	fmt.Println("    \"Order a ThinkPad for alice@example.com and send her a confirmation\"")
	fmt.Println("    \"Check if 'The Go Programming Language' is available\"")
	fmt.Println("    \"Show me all orders for alice@example.com\"")
	fmt.Println("    \"Order 3 Go Gopher t-shirts for bob@example.com, reserve the stock, and notify him\"")
	fmt.Println()

	if err := service.Run(); err != nil {
		log.Fatal(err)
	}
}
