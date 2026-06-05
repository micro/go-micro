// Package agent provides the Agent abstraction for Go Micro.
//
// An Agent is a service with an LLM inside it. It registers a Chat
// endpoint via RPC, discovers its assigned services' tools, and
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

	"go-micro.dev/v5/ai"
	"go-micro.dev/v5/server"
	"go-micro.dev/v5/store"

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
	Chat(ctx context.Context, message string) (*Response, error)
	Run() error
	Stop() error
	String() string
}

// Response is what an agent returns from Chat.
type Response struct {
	Reply     string        `json:"reply"`
	ToolCalls []ai.ToolCall `json:"tool_calls,omitempty"`
	Agent     string        `json:"agent"`
}

// ChatRequest is the RPC request for Agent.Chat.
type ChatRequest struct {
	Message string `json:"message"`
}

// ChatResponse is the RPC response for Agent.Chat.
type ChatResponse struct {
	Reply     string        `json:"reply"`
	ToolCalls []ai.ToolCall `json:"tool_calls,omitempty"`
	Agent     string        `json:"agent"`
}

type agentImpl struct {
	opts   Options
	model  ai.Model
	tools  *ai.Tools
	hist   *ai.History
	server server.Server
	mu     sync.Mutex
}

// New creates a new Agent.
func New(opts ...Option) Agent {
	return &agentImpl{
		opts: newOptions(opts...),
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
	modelOpts = append(modelOpts, ai.WithToolHandler(a.tools.Handler()))
	a.model = ai.New(a.opts.Provider, modelOpts...)

	a.hist = ai.NewHistory(a.opts.HistoryLimit)
	a.loadHistory()
}

// Chat sends a message and returns the agent's response.
func (a *agentImpl) Chat(ctx context.Context, message string) (*Response, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.model == nil {
		a.setup()
	}

	toolList, err := a.discoverTools()
	if err != nil {
		return nil, fmt.Errorf("discover tools: %w", err)
	}

	a.hist.Add("user", message)

	resp, err := a.model.Generate(ctx, &ai.Request{
		Prompt:       message,
		SystemPrompt: a.buildPrompt(),
		Tools:        toolList,
		Messages:     a.hist.Messages(),
	})
	if err != nil {
		return nil, err
	}

	if resp.Reply != "" {
		a.hist.Add("assistant", resp.Reply)
	}
	if resp.Answer != "" {
		a.hist.Add("assistant", resp.Answer)
	}

	a.saveHistory()

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

// handleChat is the RPC handler that external callers use.
func (a *agentImpl) handleChat(ctx context.Context, req *ChatRequest, rsp *ChatResponse) error {
	resp, err := a.Chat(ctx, req.Message)
	if err != nil {
		return err
	}
	rsp.Reply = resp.Reply
	rsp.ToolCalls = resp.ToolCalls
	rsp.Agent = resp.Agent
	return nil
}

// Run starts the agent as a service with a Chat RPC endpoint.
func (a *agentImpl) Run() error {
	if a.model == nil {
		a.setup()
	}

	// Create a real server with the agent's name
	a.server = server.NewServer(
		server.Name(a.opts.Name),
		server.Registry(a.opts.Registry),
		server.Metadata(map[string]string{
			"type":     "agent",
			"services": strings.Join(a.opts.Services, ","),
		}),
	)

	// Register the Chat handler
	handler := a.server.NewHandler(&agentHandler{agent: a})
	if err := a.server.Handle(handler); err != nil {
		return fmt.Errorf("failed to register handler: %w", err)
	}

	// Start the server (registers in registry, listens for RPC)
	if err := a.server.Start(); err != nil {
		return fmt.Errorf("failed to start agent server: %w", err)
	}

	fmt.Printf("Agent %s registered (manages: %s)\n", a.opts.Name, strings.Join(a.opts.Services, ", "))

	// Block until stopped
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

// agentHandler wraps the agent to expose it as an RPC service.
// The RPC endpoint is Agent.Chat.
type agentHandler struct {
	agent *agentImpl
}

// Chat handles RPC calls to Agent.Chat.
// @example {"message": "What tasks are overdue?"}
func (h *agentHandler) Chat(ctx context.Context, req *ChatRequest, rsp *ChatResponse) error {
	return h.agent.handleChat(ctx, req, rsp)
}

// discoverTools finds endpoints from the agent's assigned services,
// excluding the agent's own endpoints.
func (a *agentImpl) discoverTools() ([]ai.Tool, error) {
	all, err := a.tools.Discover()
	if err != nil {
		return nil, err
	}

	var scoped []ai.Tool
	for _, t := range all {
		// Skip our own endpoints
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
	return scoped, nil
}

func (a *agentImpl) buildPrompt() string {
	if a.opts.Prompt != "" {
		return a.opts.Prompt
	}
	if len(a.opts.Services) > 0 {
		return fmt.Sprintf("You are the %s agent. You manage these services: %s. Use the available tools to fulfill requests.",
			a.opts.Name, strings.Join(a.opts.Services, ", "))
	}
	return fmt.Sprintf("You are the %s agent. Use the available tools to fulfill requests.", a.opts.Name)
}

func (a *agentImpl) historyKey() string {
	return "agent/" + a.opts.Name + "/history"
}

func (a *agentImpl) loadHistory() {
	recs, err := a.opts.Store.Read(a.historyKey())
	if err != nil || len(recs) == 0 {
		return
	}
	var messages []ai.Message
	if err := json.Unmarshal(recs[0].Value, &messages); err != nil {
		return
	}
	for _, m := range messages {
		a.hist.Add(m.Role, m.Content)
	}
}

func (a *agentImpl) saveHistory() {
	data, err := json.Marshal(a.hist.Messages())
	if err != nil {
		return
	}
	a.opts.Store.Write(&store.Record{
		Key:   a.historyKey(),
		Value: data,
	})
}
