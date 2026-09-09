// Support desk — a real-world agent built from services, a flow, and a
// guardrail. It's the "zero to hero" shape: a few services, an agent that
// manages them, an event that triggers the agent, and a human-in-the-loop
// gate on the one action that touches a customer.
//
// The scenario: a customer files a ticket. A ticket.created event triggers
// the support agent, which looks the customer up, sets the ticket's
// priority, and drafts a reply — but it can't actually email the customer
// without passing the approval gate.
//
// Everything is real — services, registry, RPC, broker, the agent loop,
// the store. Only the LLM is mocked, so it runs with no API key. Pass
// -provider anthropic (with a key) to run it against a live model; then the
// agent reasons about the ticket itself instead of following the script.
//
// Run:
//
//	go run ./examples/support
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
// services — the support desk's capabilities
// ---------------------------------------------------------------------------

type Customer struct {
	Email string `json:"email"`
	Name  string `json:"name"`
	Plan  string `json:"plan"`
}

type LookupRequest struct {
	Email string `json:"email" description:"Customer email to look up (required)"`
}

// CustomerService is a tiny seeded customer directory.
type CustomerService struct{}

// Lookup returns the customer with the given email.
// @example {"email": "alice@acme.com"}
func (s *CustomerService) Lookup(_ context.Context, req *LookupRequest, rsp *Customer) error {
	known := map[string]Customer{
		"alice@acme.com": {Email: "alice@acme.com", Name: "Alice", Plan: "pro"},
	}
	c, ok := known[req.Email]
	if !ok {
		return fmt.Errorf("customer %s not found", req.Email)
	}
	*rsp = c
	fmt.Printf("    \033[32m[customers]\033[0m looked up %s (%s plan)\n", c.Name, c.Plan)
	return nil
}

type Ticket struct {
	ID       string `json:"id"`
	Customer string `json:"customer"`
	Subject  string `json:"subject"`
	Body     string `json:"body"`
	Priority string `json:"priority,omitempty"`
	Status   string `json:"status"`
}

type UpdateRequest struct {
	ID       string `json:"id" description:"Ticket id (required)"`
	Priority string `json:"priority" description:"Priority: low, normal, high"`
	Status   string `json:"status" description:"Status: open, in_progress, resolved"`
}

// TicketService stores support tickets in memory.
type TicketService struct {
	mu      sync.Mutex
	tickets map[string]*Ticket
}

func (s *TicketService) seed(t *Ticket) {
	s.mu.Lock()
	if s.tickets == nil {
		s.tickets = map[string]*Ticket{}
	}
	s.tickets[t.ID] = t
	s.mu.Unlock()
}

// Update changes a ticket's priority and/or status.
// @example {"id": "ticket-1", "priority": "high", "status": "in_progress"}
func (s *TicketService) Update(_ context.Context, req *UpdateRequest, rsp *Ticket) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tickets[req.ID]
	if !ok {
		return fmt.Errorf("ticket %s not found", req.ID)
	}
	if req.Priority != "" {
		t.Priority = req.Priority
	}
	if req.Status != "" {
		t.Status = req.Status
	}
	*rsp = *t
	fmt.Printf("    \033[32m[tickets]\033[0m %s → priority=%s status=%s\n", t.ID, t.Priority, t.Status)
	return nil
}

type SendRequest struct {
	To      string `json:"to" description:"Recipient email (required)"`
	Message string `json:"message" description:"Reply body (required)"`
}
type SendResponse struct {
	Sent bool `json:"sent"`
}

// NotifyService emails the customer. This is the action we gate.
type NotifyService struct{ sent int }

// Send emails a reply to a customer.
// @example {"to": "alice@acme.com", "message": "We're on it."}
func (s *NotifyService) Send(_ context.Context, req *SendRequest, rsp *SendResponse) error {
	s.sent++
	fmt.Printf("    \033[35m[notify]\033[0m 📨 to=%s: %q\n", req.To, req.Message)
	rsp.Sent = true
	return nil
}

// ---------------------------------------------------------------------------
// mock LLM — the only fake. It scripts the triage from the offered tools.
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
func (m *mockModel) Stream(context.Context, *ai.Request, ...ai.GenerateOption) (ai.Stream, error) {
	return nil, fmt.Errorf("stream not supported by mock")
}

func (m *mockModel) call(ctx context.Context, tools []ai.Tool, sub string, input map[string]any) {
	for _, t := range tools {
		if strings.Contains(t.Name, sub) && m.opts.ToolHandler != nil {
			m.opts.ToolHandler(ctx, ai.ToolCall{ID: sub, Name: t.Name, Input: input})
			return
		}
	}
}

func (m *mockModel) Generate(ctx context.Context, req *ai.Request, _ ...ai.GenerateOption) (*ai.Response, error) {
	// A real model would read the ticket from the prompt and decide. The
	// mock follows a fixed, sensible triage so the demo is deterministic.
	m.call(ctx, req.Tools, "Lookup", map[string]any{"email": "alice@acme.com"})
	m.call(ctx, req.Tools, "Update", map[string]any{"id": "ticket-1", "priority": "high", "status": "in_progress"})
	m.call(ctx, req.Tools, "Send", map[string]any{
		"to":      "alice@acme.com",
		"message": "Hi Alice — thanks for reaching out. We've bumped this to high priority and are on it.",
	})
	return &ai.Response{Answer: "Triaged ticket-1 for Alice and sent a reply."}, nil
}

// ---------------------------------------------------------------------------
// wiring
// ---------------------------------------------------------------------------

func providerKey(provider string) string {
	if v := os.Getenv("MICRO_AI_API_KEY"); v != "" {
		return v
	}
	return os.Getenv(map[string]string{
		"anthropic": "ANTHROPIC_API_KEY", "openai": "OPENAI_API_KEY",
		"gemini": "GEMINI_API_KEY", "groq": "GROQ_API_KEY", "mistral": "MISTRAL_API_KEY",
		"together": "TOGETHER_API_KEY", "atlascloud": "ATLASCLOUD_API_KEY",
	}[provider])
}

func waitFor(reg registry.Registry, names ...string) {
	deadline := time.Now().Add(5 * time.Second)
	for _, name := range names {
		for time.Now().Before(deadline) {
			if svcs, err := reg.GetService(name); err == nil && len(svcs) > 0 && len(svcs[0].Nodes) > 0 {
				break
			}
			time.Sleep(20 * time.Millisecond)
		}
	}
}

func runSupport(provider string) error {
	apiKey := ""
	if provider == "mock" {
		ai.Register("mock", newMock)
	} else if apiKey = providerKey(provider); apiKey == "" {
		return fmt.Errorf("no API key for provider %q — set MICRO_AI_API_KEY or the provider's key env", provider)
	}

	fmt.Printf("\n\033[1mSupport desk (provider: %s)\033[0m\n\n", provider)

	// Shared in-memory infrastructure so the demo runs in one process.
	reg := registry.NewMemoryRegistry()
	br := broker.NewMemoryBroker()
	if err := br.Connect(); err != nil {
		return fmt.Errorf("broker connect: %w", err)
	}
	cl := client.NewClient(client.Registry(reg), client.Selector(selector.NewSelector(selector.Registry(reg))))

	// Services.
	tickets := new(TicketService)
	notify := new(NotifyService)
	var services []service.Service
	for name, h := range map[string]any{"customers": new(CustomerService), "tickets": tickets, "notify": notify} {
		svc := service.New(service.Name(name), service.Address("127.0.0.1:0"), service.Registry(reg), service.Client(cl), service.HandleSignal(false))
		_ = svc.Handle(h)
		services = append(services, svc)
		go svc.Run()
	}
	defer func() {
		for _, svc := range services {
			_ = svc.Server().Stop()
		}
	}()

	// The support agent manages the three services. The approval gate is
	// the human-in-the-loop: it can read and triage freely, but emailing a
	// customer (notify.Send) passes through here first. Return false to hold
	// it for a person or a policy; here we approve and log.
	support := agent.New(
		agent.Name("support"),
		agent.Address("127.0.0.1:0"),
		agent.Services("customers", "tickets", "notify"),
		agent.Prompt("You are a support agent. For each ticket, look up the customer, set an "+
			"appropriate priority, and reply to them. Escalate billing issues."),
		agent.Provider(provider), agent.APIKey(apiKey),
		agent.ApproveTool(func(tool string, input map[string]any) (bool, string) {
			if strings.Contains(tool, "Send") {
				fmt.Printf("  \033[33m▣ approval gate\033[0m %s(%v) — approved\n", tool, input["to"])
			}
			return true, ""
		}),
		agent.WithRegistry(reg), agent.WithClient(cl), agent.WithStore(store.NewMemoryStore()),
	)
	go support.Run()
	defer support.Stop()

	waitFor(reg, "customers", "tickets", "notify", "support")

	// A new ticket arrives, and a flow turns the event into work for the
	// agent: the event is the prompt.
	intake := flow.New("intake",
		flow.Trigger("events.ticket.created"),
		flow.Agent("support"),
		flow.Prompt("A new support ticket arrived: {{.Data}}. Handle it."),
	)
	if err := intake.Register(reg, br, cl); err != nil {
		return fmt.Errorf("flow register: %w", err)
	}
	defer intake.Stop()

	// The customer files a ticket (it exists before the event fires).
	tickets.seed(&Ticket{ID: "ticket-1", Customer: "alice@acme.com", Subject: "Can't log in", Body: "I'm locked out.", Status: "open"})
	body, _ := json.Marshal(map[string]string{"id": "ticket-1", "customer": "alice@acme.com", "subject": "Can't log in"})

	fmt.Println("\033[1m> event:\033[0m events.ticket.created", string(body))
	fmt.Println()
	if err := br.Publish("events.ticket.created", &broker.Message{Body: body}); err != nil {
		return fmt.Errorf("publish: %w", err)
	}

	// Wait for the agent to act.
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		if notify.sent >= 1 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	if rs := intake.Results(); len(rs) > 0 {
		latest := rs[len(rs)-1]
		fmt.Printf("\n\033[1msupport agent:\033[0m %s\n", latest.Reply)
		fmt.Println("\n\033[1minspect transcript:\033[0m")
		fmt.Println("  micro inspect flow intake")
		fmt.Printf("  flow: intake runs=%d latest.reply=%q\n", len(rs), latest.Reply)
		fmt.Println("  micro agent history support")
		fmt.Printf("  agent: support runs=%d latest.status=completed\n", len(rs))
	}
	if notify.sent >= 1 {
		fmt.Println("\n\033[32m✓ ticket triaged and the customer was replied to — triggered by an event\033[0m")
		return nil
	}
	fmt.Println("\n\033[31m✗ the agent did not complete the triage\033[0m")
	return fmt.Errorf("support agent did not complete triage")
}

func main() {
	provider := flag.String("provider", "mock", "LLM provider: mock (default), anthropic, openai, ...")
	flag.Parse()

	if err := runSupport(*provider); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
