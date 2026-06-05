// Package micro is a pluggable framework for microservices
package micro

import (
	"context"

	"go-micro.dev/v5/agent"
	"go-micro.dev/v5/ai/flow"
	"go-micro.dev/v5/client"
	"go-micro.dev/v5/server"
	"go-micro.dev/v5/service"
)

type serviceKey struct{}

// Service is the interface for a go-micro service.
type Service = service.Service

// Agent is the interface for an AI agent that manages services.
type Agent = agent.Agent

// AgentOption configures an Agent.
type AgentOption = agent.Option

// Flow is an event-driven LLM orchestration unit.
type Flow = flow.Flow

// FlowOption configures a Flow.
type FlowOption = flow.Option

// Group is a set of services that share lifecycle management.
type Group = service.Group

type Option = service.Option

type Options = service.Options

// Event is used to publish messages to a topic.
type Event interface {
	// Publish publishes a message to the event topic
	Publish(ctx context.Context, msg interface{}, opts ...client.PublishOption) error
}

// Type alias to satisfy the deprecation.
type Publisher = Event

// New creates a new service with the given name and options.
//
//	service := micro.New("greeter")
//	service := micro.New("greeter", micro.Address(":8080"))
func New(name string, opts ...Option) Service {
	return service.New(append([]Option{service.Name(name)}, opts...)...)
}

// NewService creates and returns a new Service based on the packages within.
// Deprecated: Use New(name, opts...) instead.
func NewService(opts ...Option) Service {
	return service.New(opts...)
}

// NewAgent creates a new AI agent that manages the given services.
//
//	agent := micro.NewAgent("task-mgr",
//	    micro.AgentServices("task"),
//	    micro.AgentPrompt("You manage tasks."),
//	    micro.AgentProvider("anthropic"),
//	)
//	agent.Run()
func NewAgent(name string, opts ...AgentOption) Agent {
	return agent.New(append([]AgentOption{agent.Name(name)}, opts...)...)
}

// AgentServices sets which services the agent manages.
func AgentServices(names ...string) AgentOption { return agent.Services(names...) }

// AgentPrompt sets the agent's system prompt.
func AgentPrompt(p string) AgentOption { return agent.Prompt(p) }

// AgentProvider sets the LLM provider.
func AgentProvider(p string) AgentOption { return agent.Provider(p) }

// AgentModel sets the LLM model.
func AgentModel(m string) AgentOption { return agent.Model(m) }

// AgentAPIKey sets the API key for the LLM provider.
func AgentAPIKey(k string) AgentOption { return agent.APIKey(k) }

// NewFlow creates an event-driven LLM orchestration unit.
//
//	f := micro.NewFlow("onboard-user",
//	    micro.FlowTrigger("events.user.created"),
//	    micro.FlowPrompt("New user: {{.Data}}. Send welcome email."),
//	    micro.FlowProvider("anthropic"),
//	)
//	f.Register(service.Options().Registry, service.Options().Broker, service.Client())
func NewFlow(name string, opts ...FlowOption) *Flow {
	return flow.New(name, opts...)
}

// FlowTrigger sets the broker topic that triggers the flow.
func FlowTrigger(topic string) FlowOption { return flow.Trigger(topic) }

// FlowPrompt sets the prompt template. Use {{.Data}} for the event payload.
func FlowPrompt(p string) FlowOption { return flow.Prompt(p) }

// FlowProvider sets the LLM provider.
func FlowProvider(p string) FlowOption { return flow.Provider(p) }

// FlowAPIKey sets the API key for the LLM provider.
func FlowAPIKey(k string) FlowOption { return flow.APIKey(k) }

// NewGroup creates a service group for running multiple services
// in a single binary with shared lifecycle management.
func NewGroup(svcs ...Service) *Group {
	return service.NewGroup(svcs...)
}

// FromContext retrieves a Service from the Context.
func FromContext(ctx context.Context) (Service, bool) {
	s, ok := ctx.Value(serviceKey{}).(Service)
	return s, ok
}

// NewContext returns a new Context with the Service embedded within it.
func NewContext(ctx context.Context, s Service) context.Context {
	return context.WithValue(ctx, serviceKey{}, s)
}

// NewEvent creates a new event publisher.
func NewEvent(topic string, c client.Client) Event {
	if c == nil {
		c = client.NewClient()
	}

	return &event{c, topic}
}

// RegisterHandler is syntactic sugar for registering a handler.
func RegisterHandler(s server.Server, h interface{}, opts ...server.HandlerOption) error {
	return s.Handle(s.NewHandler(h, opts...))
}

// RegisterSubscriber is syntactic sugar for registering a subscriber.
func RegisterSubscriber(topic string, s server.Server, h interface{}, opts ...server.SubscriberOption) error {
	return s.Subscribe(s.NewSubscriber(topic, h, opts...))
}
