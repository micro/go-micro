// Package micro is a pluggable framework for microservices
package micro

import (
	"context"
	"time"

	"go-micro.dev/v6/agent"
	"go-micro.dev/v6/ai"
	"go-micro.dev/v6/client"
	"go-micro.dev/v6/flow"
	"go-micro.dev/v6/server"
	"go-micro.dev/v6/service"
	"go-micro.dev/v6/store"
	"go.opentelemetry.io/otel/trace"
)

type serviceKey struct{}

// Service is the interface for a go-micro service.
type Service = service.Service

// Agent is the interface for an AI agent that manages services.
type Agent = agent.Agent

// AgentResponse is what an agent returns from Ask or a resumed run.
type AgentResponse = agent.Response

// AgentStream is a stream of tool execution events followed by final-answer chunks.
type AgentStream = agent.AgentStream

// AgentOption configures an Agent.
type AgentOption = agent.Option

// Flow is an event-driven LLM orchestration unit.
type Flow = flow.Flow

// FlowRun is a checkpointed flow or agent run record.
type FlowRun = flow.Run

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

// NewService creates a new service with the given name and options. It is
// the canonical constructor, symmetric with NewAgent and NewFlow.
//
//	service := micro.NewService("greeter")
//	service := micro.NewService("greeter", micro.Address(":8080"))
func NewService(name string, opts ...Option) Service {
	return service.New(append([]Option{service.Name(name)}, opts...)...)
}

// New is a deprecated alias for NewService.
//
// Deprecated: use NewService(name, opts...) — symmetric with NewAgent and NewFlow.
func New(name string, opts ...Option) Service {
	return NewService(name, opts...)
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

// AgentBaseURL sets the base URL for the LLM provider. Use this to point
// the provider at a non-default endpoint (e.g., local Ollama, a proxy).
func AgentBaseURL(url string) AgentOption { return agent.BaseURL(url) }

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

// AgentModelCallTimeout sets the timeout for each provider Generate call.
func AgentModelCallTimeout(d time.Duration) AgentOption { return agent.ModelCallTimeout(d) }

// AgentModelRetry sets the provider retry budget and backoff for transient failures.
func AgentModelRetry(maxAttempts int, backoff time.Duration) AgentOption {
	return agent.ModelRetry(maxAttempts, backoff)
}

// AgentToolCallTimeout sets the timeout for each agent tool execution.
func AgentToolCallTimeout(d time.Duration) AgentOption { return agent.ToolCallTimeout(d) }

// AgentToolRetry sets the tool retry budget and backoff for transient failures.
// Attempts include the first call. Retries are opt-in because tools can have
// side effects; keep handlers idempotent before enabling this.
func AgentToolRetry(maxAttempts int, backoff time.Duration) AgentOption {
	return agent.ToolRetry(maxAttempts, backoff)
}

// Memory is an agent's pluggable conversation memory.
type Memory = agent.Memory

// MemoryRecall is implemented by memory backends that retrieve prior context.
type MemoryRecall = agent.MemoryRecall

// ToolFunc handles a custom agent tool call.
type ToolFunc = agent.ToolFunc

// NewMemory returns the default store-backed agent memory.
func NewMemory(s store.Store, key string, limit int) Memory { return agent.NewMemory(s, key, limit) }

// NewRetrievalMemory returns store-backed memory with bounded active context
// and durable retrieval over every prior turn.
func NewRetrievalMemory(s store.Store, key string, activeLimit int) Memory {
	return agent.NewRetrievalMemory(s, key, activeLimit)
}

// NewCompactingMemory returns store-backed memory with deterministic
// summarization and retrieval controls.
func NewCompactingMemory(s store.Store, key string, maxMessages, keepRecent int) Memory {
	return agent.NewCompactingMemory(s, key, maxMessages, keepRecent)
}

// NewInMemory returns non-persistent agent memory.
func NewInMemory(limit int) Memory { return agent.NewInMemory(limit) }

// AgentMemory sets the agent's conversation memory (default: store-backed).
func AgentMemory(m Memory) AgentOption { return agent.WithMemory(m) }

// AgentRetrievalMemory enables deterministic default-memory retrieval without
// compaction; activeLimit bounds active context while every turn is archived.
func AgentRetrievalMemory(activeLimit int) AgentOption { return agent.RetrievalMemory(activeLimit) }

// AgentCompactMemory enables deterministic default-memory compaction and
// retrieval for long-running agents.
func AgentCompactMemory(maxMessages, keepRecent int) AgentOption {
	return agent.CompactMemory(maxMessages, keepRecent)
}

// AgentMemoryRecallLimit bounds recalled archived turns injected per Ask.
func AgentMemoryRecallLimit(n int) AgentOption { return agent.MemoryRecallLimit(n) }

// AgentTool registers a custom tool the agent can call, beyond its services.
func AgentTool(name, description string, properties map[string]any, handler ToolFunc) AgentOption {
	return agent.WithTool(name, description, properties, handler)
}

// AgentA2A makes the agent serve the A2A protocol on addr (e.g. ":4000")
// when it runs, so other agents can reach it directly by URL without a
// separate gateway.
func AgentA2A(addr string) AgentOption { return agent.WithA2A(addr) }

// AgentWrapTool registers a tool-execution wrapper — the tool-side
// analog of a client/server middleware wrapper. Each wrapper takes the
// next handler and returns a new one; run code before next(...) for
// "before", after it for "after". Use it for logging, metrics, retries,
// or policy. Wrappers run outside the built-in guardrails, so they see
// every call and result, including refusals.
func AgentWrapTool(w ...ai.ToolWrapper) AgentOption {
	return agent.WrapTool(w...)
}

// AgentTraceProvider enables OpenTelemetry spans for agent runs, model calls,
// tool calls, delegation, and failures.
func AgentTraceProvider(tp trace.TracerProvider) AgentOption { return agent.TraceProvider(tp) }

// AgentWithCheckpoint sets the durability backend for agent Ask runs.
// It uses the same Checkpoint interface as flows so services, agents,
// and workflows can share one execution history backend.
func AgentWithCheckpoint(c Checkpoint) AgentOption { return agent.WithCheckpoint(c) }

// AgentPending returns checkpointed agent runs that have not completed.
// Use it at process startup to discover agent work that should be resumed.
func AgentPending(ctx context.Context, a Agent) ([]FlowRun, error) { return agent.Pending(ctx, a) }

// AgentResume resumes a checkpointed agent run by id. Completed runs return
// the persisted response without calling the model or replaying tool calls.
func AgentResume(ctx context.Context, a Agent, runID string) (*AgentResponse, error) {
	return agent.Resume(ctx, a, runID)
}

// AgentResumePending resumes every incomplete checkpointed agent run, oldest
// first. It returns the first run id that fails again so startup recovery loops
// can leave the durable backlog visible instead of swallowing the failure.
func AgentResumePending(ctx context.Context, a Agent) (string, error) {
	return agent.ResumePending(ctx, a)
}

// AgentResumeInput resumes a checkpointed agent run waiting for human input.
func AgentResumeInput(ctx context.Context, a Agent, runID, input string) (*AgentResponse, error) {
	return agent.ResumeInput(ctx, a, runID, input)
}

// AgentResumeStreamAsk resumes a checkpointed agent run by id and streams the
// resulting tool events and final answer.
func AgentResumeStreamAsk(ctx context.Context, a Agent, runID string) (AgentStream, error) {
	return agent.ResumeStreamAsk(ctx, a, runID)
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

// FlowRetryBackoff sets the delay between failed step attempts.
func FlowRetryBackoff(d time.Duration) FlowOption { return flow.RetryBackoff(d) }

// FlowWithCheckpoint sets the durability backend for stepped runs.
// Stepped flows default to a store-backed checkpoint.
func FlowWithCheckpoint(c Checkpoint) FlowOption { return flow.WithCheckpoint(c) }

// FlowDeleteOnSuccess removes a run's checkpoint on success (failed runs
// are always kept). Default: retain all.
func FlowDeleteOnSuccess() FlowOption { return flow.DeleteOnSuccess() }

// FlowTraceProvider enables OpenTelemetry spans for stepped flow runs and steps.
func FlowTraceProvider(tp trace.TracerProvider) FlowOption { return flow.TraceProvider(tp) }

// FlowCall is a step action: an RPC to a service endpoint, sending the
// state data as the request and storing the response.
func FlowCall(service, endpoint string) FlowStepFunc { return flow.Call(service, endpoint) }

// FlowLLM is a step action: one augmented-LLM turn with the services as
// tools, storing the reply.
func FlowLLM(prompt string) FlowStepFunc { return flow.LLM(prompt) }

// FlowDispatch is a step action: hand the state data to a registered
// agent over RPC, storing its reply.
func FlowDispatch(agent string) FlowStepFunc { return flow.Dispatch(agent) }

// FlowLoopOption configures a FlowLoop.
type FlowLoopOption = flow.LoopOption

// FlowLoopCondition decides whether a FlowLoop should stop, given the latest
// state and the iteration just completed.
type FlowLoopCondition = flow.LoopCondition

// FlowLoop is a step action that runs body repeatedly until a stop condition
// is met or the iteration cap is reached — the agentic loop, with a
// guaranteed ceiling so it can't run away. Compose it as a FlowStep's Run.
func FlowLoop(body FlowStepFunc, opts ...FlowLoopOption) FlowStepFunc {
	return flow.Loop(body, opts...)
}

// FlowLoopMax sets a loop's hard iteration cap — the budget guardrail.
func FlowLoopMax(n int) FlowLoopOption { return flow.LoopMax(n) }

// FlowUntil stops a loop when cond returns true (a code-defined exit).
func FlowUntil(cond FlowLoopCondition) FlowLoopOption { return flow.Until(cond) }

// FlowUntilLLM stops a loop when the flow's model judges the goal met — the
// supervised "Ralph" loop. Requires a flow model (set FlowProvider/FlowAPIKey).
func FlowUntilLLM(question string) FlowLoopOption { return flow.UntilLLM(question) }

// FlowOnIteration runs fn after each loop iteration (progress/observability).
func FlowOnIteration(fn func(iter int, state FlowState)) FlowLoopOption {
	return flow.OnIteration(fn)
}

// StoreCheckpoint returns a store-backed Checkpoint whose run keys are
// namespaced under scope (pass the flow name so each flow's runs stay in
// their own keyspace). A nil store uses the default store.
func StoreCheckpoint(s store.Store, scope string) Checkpoint {
	return flow.StoreCheckpoint(s, scope)
}

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
