package template

var (
	PubsubProtoSRV = `syntax = "proto3";

package {{dehyphen .Alias}};

option go_package = "./proto;{{dehyphen .Alias}}";

service {{title .Alias}} {
	rpc Publish(PublishRequest) returns (PublishResponse) {}
	rpc Stats(StatsRequest) returns (StatsResponse) {}
}

message Event {
	string id = 1;
	string type = 2;
	string source = 3;
	string data = 4;
	int64 timestamp = 5;
}

message PublishRequest {
	string type = 1;
	string data = 2;
}

message PublishResponse {
	string id = 1;
}

message StatsRequest {}

message StatsResponse {
	int64 published = 1;
	int64 received = 2;
}
`

	PubsubHandlerSRV = `package handler

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"go-micro.dev/v5/broker"
	log "go-micro.dev/v5/logger"

	pb "{{.Dir}}/proto"
)

const Topic = "{{lower .Alias}}.events"

type {{title .Alias}} struct {
	broker    broker.Broker
	published atomic.Int64
	received  atomic.Int64
}

func New(b broker.Broker) *{{title .Alias}} {
	return &{{title .Alias}}{broker: b}
}

// Publish sends an event to the message broker.
//
// @example {"type": "user.created", "data": "{\"id\": \"123\", \"name\": \"Alice\"}"}
func (h *{{title .Alias}}) Publish(ctx context.Context, req *pb.PublishRequest, rsp *pb.PublishResponse) error {
	event := &pb.Event{
		Id:        uuid.New().String(),
		Type:      req.Type,
		Source:    "{{lower .Alias}}",
		Data:      req.Data,
		Timestamp: time.Now().Unix(),
	}

	body, err := json.Marshal(event)
	if err != nil {
		return err
	}

	if err := h.broker.Publish(Topic, &broker.Message{Body: body}); err != nil {
		return err
	}

	h.published.Add(1)
	log.Infof("Published event %s type=%s", event.Id, event.Type)

	rsp.Id = event.Id
	return nil
}

// Stats returns the number of events published and received.
//
// @example {}
func (h *{{title .Alias}}) Stats(ctx context.Context, req *pb.StatsRequest, rsp *pb.StatsResponse) error {
	rsp.Published = h.published.Load()
	rsp.Received = h.received.Load()
	return nil
}

// Subscribe sets up a subscription to the event topic. Call this
// after the service has started.
func (h *{{title .Alias}}) Subscribe() error {
	_, err := h.broker.Subscribe(Topic, func(p broker.Event) error {
		h.received.Add(1)

		var event pb.Event
		if err := json.Unmarshal(p.Message().Body, &event); err != nil {
			log.Errorf("Failed to unmarshal event: %v", err)
			return nil
		}

		log.Infof("Received event %s type=%s data=%s", event.Id, event.Type, event.Data)
		return nil
	})
	return err
}
`

	PubsubMainSRV = `package main

import (
	"{{.Dir}}/handler"
	pb "{{.Dir}}/proto"

	"go-micro.dev/v5"
	"go-micro.dev/v5/gateway/mcp"
	log "go-micro.dev/v5/logger"
)

func main() {
	service := micro.New("{{lower .Alias}}",
		mcp.WithMCP(":3001"),
	)

	service.Init()

	h := handler.New(service.Options().Broker)
	pb.Register{{title .Alias}}Handler(service.Server(), h)

	// Subscribe to events after service starts
	go func() {
		if err := h.Subscribe(); err != nil {
			log.Fatalf("Failed to subscribe: %v", err)
		}
		log.Info("Subscribed to ", handler.Topic)
	}()

	service.Run()
}
`

	PubsubMainSRVNoMCP = `package main

import (
	"{{.Dir}}/handler"
	pb "{{.Dir}}/proto"

	"go-micro.dev/v5"
	log "go-micro.dev/v5/logger"
)

func main() {
	service := micro.New("{{lower .Alias}}")

	service.Init()

	h := handler.New(service.Options().Broker)
	pb.Register{{title .Alias}}Handler(service.Server(), h)

	go func() {
		if err := h.Subscribe(); err != nil {
			log.Fatalf("Failed to subscribe: %v", err)
		}
		log.Info("Subscribed to ", handler.Topic)
	}()

	service.Run()
}
`
)
