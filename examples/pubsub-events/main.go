// Pubsub Events example: event-driven architecture with the broker and events packages.
//
// This example shows two patterns:
//   - Broker: fire-and-forget messaging (fast, no persistence)
//   - Events: durable event streaming with replay and ack/nack
//
// No external dependencies needed — uses in-memory implementations by default.
// For production, swap in NATS (broker/nats) or NATS JetStream (events/natsjs).
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"go-micro.dev/v5"
	"go-micro.dev/v5/broker"
	"go-micro.dev/v5/events"
)

// -- Domain events --

type UserCreated struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type OrderPlaced struct {
	OrderID string  `json:"order_id"`
	UserID  string  `json:"user_id"`
	Amount  float64 `json:"amount"`
}

// -- Broker pattern: fire-and-forget --

func brokerDemo() {
	fmt.Println("=== Broker Demo (fire-and-forget) ===")
	fmt.Println()

	// Connect the broker
	if err := broker.Connect(); err != nil {
		log.Fatal(err)
	}
	defer broker.Disconnect()

	// Subscribe to user events
	sub, err := broker.Subscribe("user.created", func(e broker.Event) error {
		var user UserCreated
		if err := json.Unmarshal(e.Message().Body, &user); err != nil {
			return err
		}
		fmt.Printf("  [subscriber] Got user.created: %s (%s)\n", user.Name, user.Email)
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
	defer sub.Unsubscribe()

	// Subscribe with a queue group (load balancing across consumers)
	sub2, err := broker.Subscribe("user.created", func(e broker.Event) error {
		fmt.Printf("  [worker-group] Processing user event\n")
		return nil
	}, broker.Queue("email-workers"))
	if err != nil {
		log.Fatal(err)
	}
	defer sub2.Unsubscribe()

	// Publish events
	for i := 1; i <= 3; i++ {
		user := UserCreated{
			ID:    fmt.Sprintf("u-%d", i),
			Name:  fmt.Sprintf("User %d", i),
			Email: fmt.Sprintf("user%d@example.com", i),
		}
		body, _ := json.Marshal(user)

		if err := broker.Publish("user.created", &broker.Message{
			Header: map[string]string{"source": "pubsub-demo"},
			Body:   body,
		}); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("  [publisher] Published user.created: %s\n", user.Name)
	}

	// Give async subscribers time to process
	time.Sleep(100 * time.Millisecond)
	fmt.Println()
}

// -- Events pattern: durable streaming --

func eventsDemo() {
	fmt.Println("=== Events Demo (durable streaming) ===")
	fmt.Println()

	stream, err := events.NewStream()
	if err != nil {
		log.Fatal(err)
	}

	// Publish some order events
	orders := []OrderPlaced{
		{OrderID: "ORD-001", UserID: "u-1", Amount: 29.99},
		{OrderID: "ORD-002", UserID: "u-2", Amount: 149.50},
		{OrderID: "ORD-003", UserID: "u-1", Amount: 9.99},
	}

	for _, order := range orders {
		if err := stream.Publish("order.placed", order, events.WithMetadata(map[string]string{
			"user_id": order.UserID,
		})); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("  [publisher] Published order.placed: %s ($%.2f)\n", order.OrderID, order.Amount)
	}

	// Consume events with a consumer group
	ch, err := stream.Consume("order.placed", events.WithGroup("order-processors"))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println()
	fmt.Println("  Processing events...")

	// Read events from the channel
	timeout := time.After(500 * time.Millisecond)
	count := 0
	for {
		select {
		case ev := <-ch:
			var order OrderPlaced
			if err := ev.Unmarshal(&order); err != nil {
				log.Printf("  [consumer] unmarshal error: %v", err)
				continue
			}
			fmt.Printf("  [consumer] Received %s: order %s for user %s ($%.2f)\n",
				ev.Topic, order.OrderID, order.UserID, order.Amount)
			count++
		case <-timeout:
			fmt.Printf("\n  Processed %d events\n", count)
			return
		}
	}
}

// -- Service handler with publish --

type Notifications struct {
	broker broker.Broker
}

type NotifyRequest struct {
	UserID  string `json:"user_id"`
	Message string `json:"message"`
}

type NotifyResponse struct {
	Status string `json:"status"`
}

// Send handles notification requests and publishes an event
func (n *Notifications) Send(ctx context.Context, req *NotifyRequest, rsp *NotifyResponse) error {
	log.Printf("[notifications] Sending to user %s: %s", req.UserID, req.Message)

	// Publish a notification event for other services to consume
	body, _ := json.Marshal(map[string]string{
		"user_id": req.UserID,
		"message": req.Message,
	})

	if err := n.broker.Publish("notification.sent", &broker.Message{
		Body: body,
	}); err != nil {
		return err
	}

	rsp.Status = "sent"
	return nil
}

func main() {
	// Part 1: Broker demo (fire-and-forget)
	brokerDemo()

	// Part 2: Events demo (durable streaming)
	eventsDemo()

	// Part 3: Service with integrated publishing
	fmt.Println()
	fmt.Println("=== Service with Broker Integration ===")
	fmt.Println()
	fmt.Println("Starting notifications service on :9003")
	fmt.Println("The service publishes 'notification.sent' events when called.")
	fmt.Println()
	fmt.Println("Test with:")
	fmt.Println("  micro call notifications Notifications.Send '{\"user_id\": \"u-1\", \"message\": \"hello\"}'")

	svc := micro.New("notifications", micro.Address(":9003"))
	svc.Init()

	if err := svc.Handle(&Notifications{broker: broker.DefaultBroker}); err != nil {
		log.Fatal(err)
	}

	if err := svc.Run(); err != nil {
		log.Fatal(err)
	}
}
