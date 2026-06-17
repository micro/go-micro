// Package micro is a pluggable framework for microservices
package micro

import (
	"context"

	"go-micro.dev/v5/agent"
	"go-micro.dev/v5/ai"
	"go-micro.dev/v5/client"
	"go-micro.dev/v5/flow"
	"go-micro.dev/v5/server"
	"go-micro.dev/v5/service"
	"go-micro.dev/v5/store"
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

// ApproveFunc gates an agent's tool calls before they run.
type ApproveFunc = agent.ApproveFunc

// AgentMaxSteps bounds tool executions per Ask (0 = unbounded) — a
// stopping condition for autonomous agents.
func AgentMaxSteps(n int) AgentOption { return agent.MaxSteps(n) }

// AgentLoopLimit bounds how many times an agent may repeat the same tool
// call (same arguments) in one Ask before it's refused as a no-progress
// loop. 0 disables loop detection (defaults on).
func AgentLoopLimit(n int) AgentOption { return agent.LoopLimit(n) }

// AgentApproveTool sets a human-in-the-loop / policy hook called before
// each action the agent takes.
func AgentApproveTool(fn ApproveFunc) AgentOption { return agent.ApproveTool(fn) }

// Memory is an agent's pluggable conversation memory.
type Memory = agent.Memory

// ToolFunc handles a custom agent tool call.
type ToolFunc = agent.ToolFunc

// NewMemory returns the default store-backed agent memory.
func NewMemory(s store.Store, key string, limit int) Memory { return agent.NewMemory(s, key, limit) }

// NewInMemory returns non-persistent agent memory.
func NewInMemory(limit int) Memory { return agent.NewInMemory(limit) }

// AgentMemory sets the agent's conversation memory (default: store-backed).
func AgentMemory(m Memory) AgentOption { return agent.WithMemory(m) }

// AgentTool registers a custom tool the agent can call, beyond its services.
func AgentTool(name, description string, properties map[string]any, handler ToolFunc) AgentOption {
	return agent.WithTool(name, description, properties, handler)
}

// AgentWrapTool registers a tool-execution wrapper — the tool-side
// analogue of a client/server middleware wrapper. Each wrapper takes the
// next handler and returns a new one; run code before next(...) for
// "before", after it for "after". Use it for logging, metrics, retries,
// or policy. Wrappers run outside the built-in guardrails, so they see
// every call and result, including refusals.
func AgentWrapTool(w ...ai.ToolWrapper) AgentOption {
	return agent.WrapTool(w...)
}

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

// FlowAgent makes the flow hand each event to a named agent over RPC —
// the flow triggers, the agent reasons. Without it, the flow runs a
// single LLM step itself.
func FlowAgent(name string) FlowOption { return flow.Agent(name) }

// FlowStep is one unit of a stepped flow.
type FlowStep = flow.Step

// FlowState carries data across the steps of a flow run.
type FlowState = flow.State

// FlowStepFunc performs one step's work.
type FlowStepFunc = flow.StepFunc

// Checkpoint persists and resumes flow runs (durable execution).
type Checkpoint = flow.Checkpoint

// FlowSteps makes the flow run an ordered list of steps per event,
// checkpointed between each, instead of a single LLM turn.
func FlowSteps(steps ...FlowStep) FlowOption { return flow.Steps(steps...) }

// FlowRetry sets the flow-level retry count per step (a Step's own Retry
// overrides it).
func FlowRetry(n int) FlowOption { return flow.Retry(n) }

// FlowWithCheckpoint sets the durability backend for stepped runs.
// Stepped flows default to a store-backed checkpoint.
func FlowWithCheckpoint(c Checkpoint) FlowOption { return flow.WithCheckpoint(c) }

// FlowDeleteOnSuccess removes a run's checkpoint on success (failed runs
// are always kept). Default: retain all.
func FlowDeleteOnSuccess() FlowOption { return flow.DeleteOnSuccess() }

// FlowCall is a step action: an RPC to a service endpoint, sending the
// state data as the request and storing the response.
func FlowCall(service, endpoint string) FlowStepFunc { return flow.Call(service, endpoint) }

// FlowLLM is a step action: one augmented-LLM turn with the services as
// tools, storing the reply.
func FlowLLM(prompt string) FlowStepFunc { return flow.LLM(prompt) }

// FlowDispatch is a step action: hand the state data to a registered
// agent over RPC, storing its reply.
func FlowDispatch(agent string) FlowStepFunc { return flow.Dispatch(agent) }

// StoreCheckpoint returns a store-backed Checkpoint (nil uses the default
// store).
func StoreCheckpoint(s store.Store) Checkpoint { return flow.StoreCheckpoint(s) }

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
