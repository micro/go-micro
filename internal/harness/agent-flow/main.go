// Agent Flow harness — "the event is the prompt".
//
// No human types anything. A user.created event lands on the broker, a
// Flow renders it into a prompt and hands it to a registered agent, and
// the agent reasons and acts through its services — creating a workspace
// and sending a welcome. The whole stack is real (services, registry,
// RPC, broker, the agent loop, store); only the LLM is mocked, so it
// runs without an API key. Swap -provider to run it against a live model.
//
// Run:
//
//	go run ./internal/harness/agent-flow
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
	"go-micro.dev/v6/flow"
	"go-micro.dev/v6/internal/harness/harnessutil"
	"go-micro.dev/v6/registry"
	"go-micro.dev/v6/service"
	"go-micro.dev/v6/store"
)

// ---------------------------------------------------------------------------
// real services
// ---------------------------------------------------------------------------

type Workspace struct {
	ID    string `json:"id"`
	Owner string `json:"owner"`
}

type CreateRequest struct {
	Owner string `json:"owner" description:"Owner email of the workspace (required)"`
}
type CreateResponse struct {
	Workspace *Workspace `json:"workspace"`
}

type WorkspaceService struct {
	mu sync.Mutex
	n  int
}

// Create provisions a workspace for a new user.
// @example {"owner": "alice@acme.com"}
func (s *WorkspaceService) Create(ctx context.Context, req *CreateRequest, rsp *CreateResponse) error {
	s.mu.Lock()
	s.n++
	id := fmt.Sprintf("ws-%d", s.n)
	s.mu.Unlock()
	fmt.Printf("    \033[32m[workspace]\033[0m created %s for %s\n", id, req.Owner)
	rsp.Workspace = &Workspace{ID: id, Owner: req.Owner}
	return nil
}

func (s *WorkspaceService) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.n
}

type SendRequest struct {
	To      string `json:"to" description:"Recipient address (required)"`
	Message string `json:"message" description:"Message body (required)"`
}
type SendResponse struct {
	Sent bool `json:"sent"`
}
type NotifyService struct {
	mu sync.Mutex
	n  int
}

// Send delivers a notification message to a recipient.
// @example {"to": "alice@acme.com", "message": "Welcome"}
func (s *NotifyService) Send(ctx context.Context, req *SendRequest, rsp *SendResponse) error {
	s.mu.Lock()
	s.n++
	s.mu.Unlock()
	fmt.Printf("    \033[35m[notify]\033[0m 📨 to=%s message=%q\n", req.To, req.Message)
	rsp.Sent = true
	return nil
}

func (s *NotifyService) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.n
}

// ---------------------------------------------------------------------------
// mock LLM — the only fake. It reasons by the tools it's offered.
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

func findTool(tools []ai.Tool, sub string) string {
	for _, t := range tools {
		if strings.Contains(t.Name, sub) {
			return t.Name
		}
	}
	return ""
}

func (m *mockModel) call(name string, input map[string]any) {
	args, _ := json.Marshal(input)
	fmt.Printf("  \033[33m[onboarder]\033[0m → %s(%s)\n", name, args)
	if m.opts.ToolHandler != nil {
		m.opts.ToolHandler(context.Background(), ai.ToolCall{Name: name, Input: input})
	}
}

func (m *mockModel) Generate(ctx context.Context, req *ai.Request, _ ...ai.GenerateOption) (*ai.Response, error) {
	owner := "alice@acme.com"
	if create := findTool(req.Tools, "Create"); create != "" {
		m.call(create, map[string]any{"owner": owner})
	}
	if send := findTool(req.Tools, "Send"); send != "" {
		m.call(send, map[string]any{"to": owner, "message": "Welcome — your workspace is ready."})
	}
	return &ai.Response{Answer: "Onboarded " + owner + "."}, nil
}

// ---------------------------------------------------------------------------
// wiring
// ---------------------------------------------------------------------------

func providerKey(provider string) string {
	if v := os.Getenv("MICRO_AI_API_KEY"); v != "" {
		return v
	}
	env := map[string]string{
		"anthropic": "ANTHROPIC_API_KEY", "openai": "OPENAI_API_KEY",
		"gemini": "GEMINI_API_KEY", "groq": "GROQ_API_KEY", "mistral": "MISTRAL_API_KEY",
		"together": "TOGETHER_API_KEY", "atlascloud": "ATLASCLOUD_API_KEY",
	}[provider]
	return os.Getenv(env)
}

func waitFor(reg registry.Registry, name string) {
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if svcs, err := reg.GetService(name); err == nil && len(svcs) > 0 && len(svcs[0].Nodes) > 0 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func waitForOnboardingSideEffects(ctx context.Context, wsSvc *WorkspaceService, ntSvc *NotifyService) error {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		workspaces, notifications := wsSvc.count(), ntSvc.count()
		if workspaces >= 1 && notifications >= 1 {
			return nil
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("agent-flow missing required onboarding side effects before timeout: workspaces=%d/1 notifications=%d/1", workspaces, notifications)
		case <-ticker.C:
		}
	}
}

func main() {
	provider := flag.String("provider", "mock", "LLM provider: mock (default), anthropic, openai, ...")
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

	fmt.Printf("\n\033[1mAgent Flow — the event is the prompt (provider: %s)\033[0m\n", *provider)
	fmt.Print("No human prompt: a user.created event triggers an agent that onboards the user.\n\n")

	reg := registry.NewMemoryRegistry()
	br := broker.NewMemoryBroker()
	if err := br.Connect(); err != nil {
		fmt.Println("broker connect:", err)
		os.Exit(1)
	}
	cl := harnessutil.Client(*provider, reg)
	mem := store.NewMemoryStore()
	liveAgentOpts := harnessutil.AgentOptions(*provider)

	wsSvc := new(WorkspaceService)
	ws := service.New(service.Name("workspace"), service.Address("127.0.0.1:0"), service.Registry(reg), service.Client(cl))
	ws.Handle(wsSvc)
	go ws.Run()

	ntSvc := new(NotifyService)
	nt := service.New(service.Name("notify"), service.Address("127.0.0.1:0"), service.Registry(reg), service.Client(cl))
	nt.Handle(ntSvc)
	go nt.Run()

	// The onboarder agent, registered so the flow can reach it over RPC.
	onboarderOpts := []agent.Option{
		agent.Name("onboarder"),
		agent.Address("127.0.0.1:0"),
		agent.Services("workspace", "notify"),
		agent.Prompt("You onboard new users. Create their workspace and send a welcome message."),
		agent.Provider(*provider),
		agent.APIKey(apiKey),
		agent.WithRegistry(reg), agent.WithClient(cl), agent.WithStore(mem),
	}
	onboarderOpts = append(onboarderOpts, liveAgentOpts...)
	onboarder := agent.New(onboarderOpts...)
	go onboarder.Run()
	defer onboarder.Stop()

	waitFor(reg, "workspace")
	waitFor(reg, "notify")
	waitFor(reg, "onboarder")

	// A workflow that turns the event into a prompt for the agent.
	f := flow.New("onboard",
		flow.Trigger("events.user.created"),
		flow.Agent("onboarder"),
		flow.Prompt("A new user signed up: {{.Data}}. Get them set up."),
		flow.Timeout(harnessutil.LiveTimeout(*provider)),
	)
	if err := f.Register(reg, br, cl); err != nil {
		fmt.Println("flow register:", err)
		os.Exit(1)
	}

	fmt.Print("\033[1m> event:\033[0m publishing events.user.created {\"email\":\"alice@acme.com\"}\n\n")

	// The event — no human in the loop.
	if err := br.Publish("events.user.created", &broker.Message{
		Body: []byte(`{"email":"alice@acme.com"}`),
	}); err != nil {
		fmt.Println("publish:", err)
		os.Exit(1)
	}

	// Wait for the agent to finish acting, and fail the harness if the
	// provider returns a successful reply without the required service side
	// effects. The 0→hero/provider conformance path must not print success
	// unless the services → agent → workflow contract actually happened.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	err := waitForOnboardingSideEffects(ctx, wsSvc, ntSvc)
	cancel()

	fmt.Printf("\n\033[1mresult:\033[0m workspaces created=%d, notifications sent=%d\n", wsSvc.count(), ntSvc.count())
	if rs := f.Results(); len(rs) > 0 {
		fmt.Printf("flow reply: %s\n", rs[len(rs)-1].Reply)
	}
	if err != nil {
		fmt.Printf("\n\033[31m✗ %v\033[0m\n", err)
		os.Exit(1)
	}
	fmt.Println("\n\033[32m✓ the agent onboarded the user — triggered by an event, not a prompt\033[0m")
}
