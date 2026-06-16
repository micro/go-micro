// Package agent provides the Agent abstraction for Go Micro.
//
// An Agent is a service with an LLM inside it. It registers a Chat
// RPC endpoint, discovers its assigned services' tools, and
// orchestrates them intelligently.
//
//	agent := micro.NewAgent("task-mgr",
//	    micro.AgentServices("task"),
//	    micro.AgentPrompt("You manage tasks."),
//	    micro.AgentProvider("anthropic"),
//	)
//	agent.Run()
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	pb "go-micro.dev/v5/agent/proto"
	"go-micro.dev/v5/ai"
	"go-micro.dev/v5/server"

	_ "go-micro.dev/v5/ai/anthropic"
	_ "go-micro.dev/v5/ai/atlascloud"
	_ "go-micro.dev/v5/ai/gemini"
	_ "go-micro.dev/v5/ai/groq"
	_ "go-micro.dev/v5/ai/mistral"
	_ "go-micro.dev/v5/ai/openai"
	_ "go-micro.dev/v5/ai/together"
)

// Agent is the interface for an AI agent that manages services.
type Agent interface {
	Name() string
	Init(...Option)
	Options() Options
	Ask(ctx context.Context, message string) (*Response, error)
	Run() error
	Stop() error
	String() string
}

// Response is what an agent returns from Chat.
type Response struct {
	Reply     string
	ToolCalls []ai.ToolCall
	Agent     string
}

type agentImpl struct {
	opts   Options
	model  ai.Model
	tools  *ai.Tools
	mem    Memory
	server server.Server
	mu     sync.Mutex

	// ephemeral marks a short-lived sub-agent created by delegation.
	// Ephemeral agents run with an isolated context: they load and
	// persist no history, and have no built-in tools (so they cannot
	// plan or re-delegate).
	ephemeral bool

	// steps counts tool executions in the current Ask, for MaxSteps.
	steps int
	// calls counts identical tool calls (name+args) in the current Ask,
	// for LoopLimit.
	calls map[string]int
}

// New creates a new Agent.
func New(opts ...Option) Agent {
	return &agentImpl{
		opts: newOptions(opts...),
	}
}

// newEphemeral creates a short-lived sub-agent for a delegated subtask.
// It shares the parent's provider, model, and infrastructure but runs
// with an isolated context: it loads and persists no history and has no
// built-in tools (so it can neither plan nor re-delegate). Returns the
// concrete type because ephemeral is an internal construction detail,
// not a public option.
func newEphemeral(opts ...Option) *agentImpl {
	return &agentImpl{
		opts:      newOptions(opts...),
		ephemeral: true,
	}
}

func (a *agentImpl) Name() string {
	return a.opts.Name
}

func (a *agentImpl) Init(opts ...Option) {
	for _, o := range opts {
		o(&a.opts)
	}
	a.setup()
}

func (a *agentImpl) Options() Options {
	return a.opts
}

func (a *agentImpl) String() string {
	return "agent"
}

func (a *agentImpl) setup() {
	var modelOpts []ai.Option
	modelOpts = append(modelOpts, ai.WithAPIKey(a.opts.APIKey))
	if a.opts.Model != "" {
		modelOpts = append(modelOpts, ai.WithModel(a.opts.Model))
	}

	a.tools = ai.NewTools(a.opts.Registry, ai.ToolClient(a.opts.Client))
	modelOpts = append(modelOpts, ai.WithToolHandler(a.toolHandler()))
	a.model = ai.New(a.opts.Provider, modelOpts...)

	// Memory is pluggable. Use the configured one, otherwise the default
	// store-backed memory — except ephemeral sub-agents, which keep an
	// isolated, non-persistent context.
	switch {
	case a.opts.Memory != nil:
		a.mem = a.opts.Memory
	case a.ephemeral:
		a.mem = NewInMemory(a.opts.HistoryLimit)
	default:
		a.mem = NewMemory(a.opts.Store, "agent/"+a.opts.Name+"/history", a.opts.HistoryLimit)
	}
}

// Ask sends a message and returns the agent's response.
// This is the programmatic API for direct use.
func (a *agentImpl) Ask(ctx context.Context, message string) (*Response, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.model == nil {
		a.setup()
	}

	toolList, err := a.discoverTools()
	if err != nil {
		return nil, fmt.Errorf("discover tools: %w", err)
	}

	a.mem.Add("user", message)
	a.steps = 0
	a.calls = map[string]int{}

	resp, err := a.model.Generate(ctx, &ai.Request{
		Prompt:       message,
		SystemPrompt: a.buildPrompt(),
		Tools:        toolList,
		Messages:     a.mem.Messages(),
	})
	if err != nil {
		return nil, err
	}

	if resp.Reply != "" {
		a.mem.Add("assistant", resp.Reply)
	}
	if resp.Answer != "" {
		a.mem.Add("assistant", resp.Answer)
	}

	reply := resp.Reply
	if resp.Answer != "" {
		if reply != "" {
			reply += "\n\n"
		}
		reply += resp.Answer
	}

	return &Response{
		Reply:     reply,
		ToolCalls: resp.ToolCalls,
		Agent:     a.opts.Name,
	}, nil
}

// Chat implements the proto AgentHandler interface for RPC.
// @example {"message": "What tasks are overdue?"}
func (a *agentImpl) Chat(ctx context.Context, req *pb.ChatRequest, rsp *pb.ChatResponse) error {
	resp, err := a.Ask(ctx, req.Message)
	if err != nil {
		return err
	}
	rsp.Reply = resp.Reply
	rsp.Agent = resp.Agent
	for _, tc := range resp.ToolCalls {
		input, _ := json.Marshal(tc.Input)
		rsp.ToolCalls = append(rsp.ToolCalls, &pb.ToolCall{
			Id:     tc.ID,
			Name:   tc.Name,
			Input:  string(input),
			Result: tc.Result,
		})
	}
	return nil
}

// Run starts the agent as a service with a Chat RPC endpoint.
func (a *agentImpl) Run() error {
	if a.model == nil {
		a.setup()
	}

	a.server = server.NewServer(
		server.Name(a.opts.Name),
		server.Registry(a.opts.Registry),
		server.Metadata(map[string]string{
			"type":     "agent",
			"services": strings.Join(a.opts.Services, ","),
		}),
	)

	pb.RegisterAgentHandler(a.server, a)

	if err := a.server.Start(); err != nil {
		return fmt.Errorf("failed to start agent: %w", err)
	}

	fmt.Printf("Agent %s registered (manages: %s)\n", a.opts.Name, strings.Join(a.opts.Services, ", "))

	ch := make(chan struct{})
	<-ch
	return nil
}

func (a *agentImpl) Stop() error {
	if a.server != nil {
		return a.server.Stop()
	}
	return nil
}

func (a *agentImpl) discoverTools() ([]ai.Tool, error) {
	all, err := a.tools.Discover()
	if err != nil {
		return nil, err
	}

	var scoped []ai.Tool
	for _, t := range all {
		if strings.HasPrefix(t.OriginalName, a.opts.Name+".") {
			continue
		}
		if len(a.opts.Services) == 0 {
			scoped = append(scoped, t)
			continue
		}
		for _, svc := range a.opts.Services {
			if strings.HasPrefix(t.OriginalName, svc+".") {
				scoped = append(scoped, t)
				break
			}
		}
	}

	// Developer-registered custom tools (WithTool).
	for i := range a.opts.tools {
		scoped = append(scoped, a.opts.tools[i].def)
	}

	// Expose the agent's own capabilities (plan, delegate) as tools.
	// Ephemeral sub-agents don't get them.
	if !a.ephemeral {
		scoped = append(scoped, builtinTools()...)
	}
	return scoped, nil
}

func (a *agentImpl) buildPrompt() string {
	var base string
	switch {
	case a.opts.Prompt != "":
		base = a.opts.Prompt
	case len(a.opts.Services) > 0:
		base = fmt.Sprintf("You are the %s agent. You manage these services: %s. Use the available tools to fulfill requests.",
			a.opts.Name, strings.Join(a.opts.Services, ", "))
	default:
		base = fmt.Sprintf("You are the %s agent. Use the available tools to fulfill requests.", a.opts.Name)
	}

	// Keep the agent oriented: surface its saved plan, if any.
	if !a.ephemeral {
		if plan := a.loadPlan(); plan != "" {
			base += "\n\nYour current plan (update it with the plan tool as you make progress):\n" + plan
		}
	}
	return base
}
