// Plan & Delegate integration harness.
//
// This runs the REAL go-micro stack end to end — real services, real
// registry, real RPC, the real agent loop, real store, real delegate
// routing — and mocks ONLY the LLM with a deterministic provider. It
// proves the plumbing works without an API key, and it's reproducible.
//
// Swap MICRO_AI_PROVIDER/MICRO_AI_API_KEY (and remove --mock) to run the
// exact same flow against a live model.
//
// Run:
//
//	go run ./internal/harness/plan-delegate
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"go-micro.dev/v6/agent"
	"go-micro.dev/v6/ai"
	"go-micro.dev/v6/broker"
	"go-micro.dev/v6/client"
	"go-micro.dev/v6/flow"
	"go-micro.dev/v6/registry"
	"go-micro.dev/v6/selector"
	"go-micro.dev/v6/service"
	"go-micro.dev/v6/store"
)

// ---------------------------------------------------------------------------
// real services
// ---------------------------------------------------------------------------

type Task struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type AddRequest struct {
	Title string `json:"title" description:"Title of the task to add"`
}
type AddResponse struct {
	Task *Task `json:"task"`
}
type ListRequest struct{}
type ListResponse struct {
	Tasks []*Task `json:"tasks"`
}

type TaskService struct {
	mu     sync.Mutex
	tasks  []*Task
	nextID int
}

// Add creates a new task with the given title.
// @example {"title": "Design"}
func (s *TaskService) Add(ctx context.Context, req *AddRequest, rsp *AddResponse) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	t := &Task{ID: fmt.Sprintf("task-%d", s.nextID), Title: req.Title}
	s.tasks = append(s.tasks, t)
	rsp.Task = t
	fmt.Printf("    \033[32m[task]\033[0m created %s %q\n", t.ID, t.Title)
	return nil
}

// List returns all tasks.
// @example {}
func (s *TaskService) List(ctx context.Context, req *ListRequest, rsp *ListResponse) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	rsp.Tasks = append(rsp.Tasks, s.tasks...)
	return nil
}

func (s *TaskService) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.tasks)
}

type SendRequest struct {
	To      string `json:"to" description:"Recipient address"`
	Message string `json:"message" description:"Message body"`
}
type SendResponse struct {
	Sent bool `json:"sent"`
}
type NotifyService struct {
	mu   sync.Mutex
	sent int
}

// Send delivers a notification message to a recipient.
// @example {"to": "owner@acme.com", "message": "ready"}
func (s *NotifyService) Send(ctx context.Context, req *SendRequest, rsp *SendResponse) error {
	s.mu.Lock()
	s.sent++
	s.mu.Unlock()
	fmt.Printf("    \033[35m[notify]\033[0m 📨 to=%s message=%q\n", req.To, req.Message)
	rsp.Sent = true
	return nil
}

func (s *NotifyService) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sent
}

// ---------------------------------------------------------------------------
// mock LLM provider — the ONLY fake. It "reasons" by simple heuristics
// over the tools it's offered and the system prompt it's given, calling
// the real tool handler exactly the way a real provider would.
// ---------------------------------------------------------------------------

type mockModel struct{ opts ai.Options }

func newMock(opts ...ai.Option) ai.Model {
	m := &mockModel{}
	_ = m.Init(opts...)
	return m
}

func (m *mockModel) Init(opts ...ai.Option) error {
	for _, o := range opts {
		o(&m.opts)
	}
	return nil
}
func (m *mockModel) Options() ai.Options { return m.opts }
func (m *mockModel) String() string      { return "mock" }
func (m *mockModel) Stream(ctx context.Context, req *ai.Request, _ ...ai.GenerateOption) (ai.Stream, error) {
	return nil, fmt.Errorf("stream not supported by mock")
}

// findTool returns the safe name of the first offered tool whose name
// contains sub, or "" if none.
func findTool(tools []ai.Tool, sub string) string {
	for _, t := range tools {
		if strings.Contains(t.Name, sub) {
			return t.Name
		}
	}
	return ""
}

func (m *mockModel) call(who, name string, input map[string]any) {
	args, _ := json.Marshal(input)
	fmt.Printf("  \033[33m[%s]\033[0m → %s(%s)\n", who, name, args)
	if m.opts.ToolHandler != nil {
		m.opts.ToolHandler(context.Background(), ai.ToolCall{Name: name, Input: input})
	}
}

func (m *mockModel) Generate(ctx context.Context, req *ai.Request, _ ...ai.GenerateOption) (*ai.Response, error) {
	// Classify by the tools actually offered, not by prompt text:
	// the conductor has the task "Add" tool, comms has "Send".
	hasAdd := findTool(req.Tools, "Add") != ""
	hasSend := findTool(req.Tools, "Send") != ""

	switch {
	// comms agent: owns notify, has Send but not Add.
	case hasSend && !hasAdd:
		send := findTool(req.Tools, "Send")
		m.call("comms", send, map[string]any{
			"to":      "owner@acme.com",
			"message": "The launch plan is ready",
		})
		return &ai.Response{Answer: "Notified owner@acme.com."}, nil

	// conductor: has the task Add tool — plan, create tasks, delegate.
	case hasAdd:
		if plan := findTool(req.Tools, "plan"); plan != "" {
			m.call("conductor", plan, map[string]any{
				"steps": []any{
					map[string]any{"task": "create Design task", "status": "pending"},
					map[string]any{"task": "create Build task", "status": "pending"},
					map[string]any{"task": "create Ship task", "status": "pending"},
					map[string]any{"task": "notify owner via comms", "status": "pending"},
				},
			})
		}
		if add := findTool(req.Tools, "Add"); add != "" {
			for _, title := range []string{"Design", "Build", "Ship"} {
				m.call("conductor", add, map[string]any{"title": title})
			}
		}
		if del := findTool(req.Tools, "delegate"); del != "" {
			m.call("conductor", del, map[string]any{
				"task": "Notify owner@acme.com that the launch plan is ready",
				"to":   "comms",
			})
		}
		return &ai.Response{Answer: "Created Design, Build and Ship, and had comms notify the owner."}, nil

	// ephemeral sub-agent or anything else.
	default:
		return &ai.Response{Reply: "subtask handled"}, nil
	}
}

func providerKey(provider string) string {
	if v := os.Getenv("MICRO_AI_API_KEY"); v != "" {
		return v
	}
	env := map[string]string{
		"anthropic":  "ANTHROPIC_API_KEY",
		"openai":     "OPENAI_API_KEY",
		"gemini":     "GEMINI_API_KEY",
		"groq":       "GROQ_API_KEY",
		"mistral":    "MISTRAL_API_KEY",
		"together":   "TOGETHER_API_KEY",
		"atlascloud": "ATLASCLOUD_API_KEY",
	}[provider]
	return os.Getenv(env)
}

func main() {
	provider := flag.String("provider", "mock", "LLM provider: mock (default), anthropic, openai, gemini, groq, mistral, together, atlascloud")
	flag.Parse()

	apiKey := ""
	if *provider == "mock" {
		ai.Register("mock", newMock)
	} else {
		apiKey = providerKey(*provider)
		if apiKey == "" {
			fmt.Printf("no API key for provider %q — set MICRO_AI_API_KEY or the provider's key env\n", *provider)
			return
		}
	}

	fmt.Printf("\n\033[1mPlan & Delegate — live integration harness (provider: %s)\033[0m\n", *provider)
	fmt.Print("Real services, registry, RPC, agent loop, store, delegation.\n\n")

	reg := registry.NewMemoryRegistry()
	cl := client.NewClient(client.Registry(reg), client.Selector(selector.NewSelector(selector.Registry(reg))))
	mem := store.NewMemoryStore()

	// Real services.
	taskSvc := new(TaskService)
	task := service.New(service.Name("task"), service.Address("127.0.0.1:0"), service.Registry(reg), service.Client(cl))
	if err := task.Handle(taskSvc); err != nil {
		fmt.Println("task handle:", err)
		os.Exit(1)
	}
	go task.Run()

	notifySvc := new(NotifyService)
	notify := service.New(service.Name("notify"), service.Address("127.0.0.1:0"), service.Registry(reg), service.Client(cl))
	if err := notify.Handle(notifySvc); err != nil {
		fmt.Println("notify handle:", err)
		os.Exit(1)
	}
	go notify.Run()

	// Real comms agent (owns notify), registered so delegate reaches it over RPC.
	comms := agent.New(
		agent.Name("comms"),
		agent.Address("127.0.0.1:0"),
		agent.Services("notify"),
		agent.Prompt("You handle outbound notifications. Use the notify service."),
		agent.Provider(*provider), agent.APIKey(apiKey),
		agent.WithRegistry(reg), agent.WithClient(cl), agent.WithStore(mem),
	)
	go comms.Run()
	defer comms.Stop()

	// Real conductor agent (owns task), registered so the flow can reach it over RPC.
	conductor := agent.New(
		agent.Name("conductor"),
		agent.Address("127.0.0.1:0"),
		agent.Services("task"),
		agent.Prompt("You coordinate launch work. Plan first, create tasks, and delegate notifications to the \"comms\" agent."),
		agent.Provider(*provider), agent.APIKey(apiKey),
		agent.WithRegistry(reg), agent.WithClient(cl), agent.WithStore(mem),
	)
	go conductor.Run()
	defer conductor.Stop()

	fmt.Println("waiting for services + agents to register...")
	waitForService := func(name string) {
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			if svcs, err := reg.GetService(name); err == nil && len(svcs) > 0 && len(svcs[0].Nodes) > 0 {
				return
			}
			time.Sleep(20 * time.Millisecond)
		}
	}
	for _, name := range []string{"task", "notify", "comms", "conductor"} {
		waitForService(name)
	}

	f := flow.New("zero-to-hero",
		flow.Agent("conductor"),
		flow.Prompt("Create three launch tasks (Design, Build, Ship), then make sure owner@acme.com is notified: {{.Data}}"),
	)
	if err := f.Register(reg, broker.DefaultBroker, cl); err != nil {
		fmt.Println("flow register:", err)
		os.Exit(1)
	}

	fmt.Print("\n\033[1m> flow:\033[0m services + agents + workflow + plan/delegate, no API key.\n\n")
	if err := f.Execute(context.Background(), "launch readiness"); err != nil {
		fmt.Println("\033[31merror:\033[0m", err)
		os.Exit(1)
	}

	if rs := f.Results(); len(rs) > 0 {
		fmt.Println("\n\033[1m< conductor reply:\033[0m", rs[len(rs)-1].Reply)
	}

	// Prove plan was persisted to the real store.
	if recs, _ := store.Scope(mem, "agent", "conductor").Read("plan"); len(recs) > 0 {
		fmt.Printf("\n\033[1mstored plan (agent/conductor/plan):\033[0m %s\n", string(recs[0].Value))
	} else {
		fmt.Println("\n\033[31m! plan was not persisted\033[0m")
		os.Exit(1)
	}
	if taskSvc.count() != 3 || notifySvc.count() != 1 {
		fmt.Printf("\n\033[31m! unexpected side effects: tasks=%d notify=%d\033[0m\n", taskSvc.count(), notifySvc.count())
		os.Exit(1)
	}

	fmt.Println("\n\033[32m✓ 0→hero flow complete (services → agents → workflow)\033[0m")
}
