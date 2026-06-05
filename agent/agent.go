// Package agent provides the Agent abstraction for Go Micro.
//
// An Agent is an intelligent layer that manages one or more services.
// It sits alongside Service as a top-level abstraction:
//
//	service := micro.New("task")          // capability
//	agent := micro.NewAgent("task-mgr")   // intelligence
//
// The service doesn't know about its agent. The agent discovers its
// services from the registry, scopes its tools to their endpoints,
// and maintains conversation memory in the store.
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"go-micro.dev/v5/ai"
	"go-micro.dev/v5/broker"
	"go-micro.dev/v5/registry"
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
	Reply     string
	ToolCalls []ai.ToolCall
	Agent     string
}

type agentImpl struct {
	opts  Options
	model ai.Model
	tools *ai.Tools
	hist  *ai.History
	mu    sync.Mutex
	stop  chan struct{}
}

// New creates a new Agent.
func New(opts ...Option) Agent {
	a := &agentImpl{
		opts: newOptions(opts...),
		stop: make(chan struct{}),
	}
	return a
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
	// Create LLM model
	var modelOpts []ai.Option
	modelOpts = append(modelOpts, ai.WithAPIKey(a.opts.APIKey))
	if a.opts.Model != "" {
		modelOpts = append(modelOpts, ai.WithModel(a.opts.Model))
	}

	// Create tools for service discovery + RPC
	a.tools = ai.NewTools(a.opts.Registry, ai.ToolClient(a.opts.Client))

	// Wrap the tool handler so the model can execute calls
	modelOpts = append(modelOpts, ai.WithToolHandler(a.tools.Handler()))

	a.model = ai.New(a.opts.Provider, modelOpts...)

	// Load history from store or start fresh
	a.hist = ai.NewHistory(a.opts.HistoryLimit)
	a.loadHistory()
}

func (a *agentImpl) Chat(ctx context.Context, message string) (*Response, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.model == nil {
		a.setup()
	}

	// Discover and scope tools to assigned services
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

func (a *agentImpl) Run() error {
	if a.model == nil {
		a.setup()
	}

	// Register agent in the registry
	nodeID := a.opts.Name + "-" + fmt.Sprintf("%d", os.Getpid())
	svc := &registry.Service{
		Name: a.opts.Name,
		Metadata: map[string]string{
			"type":     "agent",
			"services": strings.Join(a.opts.Services, ","),
		},
		Nodes: []*registry.Node{
			{
				Id:      nodeID,
				Address: fmt.Sprintf("127.0.0.1:%d", 40000+os.Getpid()%10000),
				Metadata: map[string]string{
					"type":     "agent",
					"services": strings.Join(a.opts.Services, ","),
				},
			},
		},
	}

	if err := a.opts.Registry.Register(svc); err != nil {
		return fmt.Errorf("failed to register agent: %w", err)
	}
	defer a.opts.Registry.Deregister(svc)

	fmt.Printf("Agent %s registered (manages: %s)\n", a.opts.Name, strings.Join(a.opts.Services, ", "))

	// Try to subscribe to agent messages on the broker
	a.opts.Broker.Connect()
	sub, err := a.opts.Broker.Subscribe("agent."+a.opts.Name, func(p broker.Event) error {
		msg := p.Message()
		if msg == nil || len(msg.Body) == 0 {
			return nil
		}
		ctx := context.Background()
		resp, err := a.Chat(ctx, string(msg.Body))
		if err != nil {
			return err
		}
		// If there's a reply-to, publish the response
		if replyTo := msg.Header["reply-to"]; replyTo != "" {
			body, _ := json.Marshal(resp)
			a.opts.Broker.Publish(replyTo, &broker.Message{Body: body})
		}
		return nil
	})
	if err == nil {
		defer sub.Unsubscribe()
	}

	// Block until signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	select {
	case <-sigCh:
	case <-a.stop:
	}

	return nil
}

func (a *agentImpl) Stop() error {
	select {
	case <-a.stop:
	default:
		close(a.stop)
	}
	return nil
}

// discoverTools finds endpoints from the agent's assigned services.
func (a *agentImpl) discoverTools() ([]ai.Tool, error) {
	all, err := a.tools.Discover()
	if err != nil {
		return nil, err
	}

	// If no services specified, return all (unscoped agent)
	if len(a.opts.Services) == 0 {
		return all, nil
	}

	var scoped []ai.Tool
	for _, t := range all {
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

// Memory persistence

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
