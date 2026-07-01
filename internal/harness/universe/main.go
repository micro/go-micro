// Universe harness — a mini end-to-end world, spun up and shut down.
//
// It boots a small but real go-micro system in one process and drives a
// realistic scenario across all the moving parts:
//
//   - four real services (inventory, payment, orders, notify) over RPC;
//   - a DURABLE FLOW "checkout" with ordered, checkpointed steps that
//     reserves, charges, confirms, then hands off to an agent;
//   - a CRASH + RESUME: the payment dependency fails on first contact, the
//     run is checkpointed at that step, and on resume it continues without
//     re-running the steps that already completed;
//   - an AGENT "concierge" with guardrails and a tool-execution wrapper,
//     reached by the flow over RPC, that sends the welcome notification;
//   - SCOPED STATE: the flow's runs and the agent's history land in their
//     own store tables.
//
// Everything is real — registry, RPC, broker, store, the flow engine, the
// agent loop. Only the LLM is mocked, so it runs free and deterministically
// in CI. Pass -provider anthropic (with a key) to run it against a live
// model. It exits non-zero if any assertion fails, so it doubles as an
// end-to-end test.
//
// Run:
//
//	go run ./internal/harness/universe
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"go-micro.dev/v6/agent"
	"go-micro.dev/v6/ai"
	"go-micro.dev/v6/broker"
	"go-micro.dev/v6/flow"
	"go-micro.dev/v6/gateway/a2a"
	"go-micro.dev/v6/internal/harness/harnessutil"
	"go-micro.dev/v6/registry"
	"go-micro.dev/v6/service"
	"go-micro.dev/v6/store"
)

// ---------------------------------------------------------------------------
// services
// ---------------------------------------------------------------------------

type Order struct {
	Order     string `json:"order" description:"Order id (required)"`
	Reserved  bool   `json:"reserved,omitempty"`
	Charged   bool   `json:"charged,omitempty"`
	Confirmed bool   `json:"confirmed,omitempty"`
}

type Inventory struct{ reserves int64 }

// Reserve holds stock for an order.
// @example {"order": "order-1"}
func (s *Inventory) Reserve(_ context.Context, req *Order, rsp *Order) error {
	atomic.AddInt64(&s.reserves, 1)
	fmt.Printf("    \033[32m[inventory]\033[0m reserved %s\n", req.Order)
	*rsp = *req
	rsp.Reserved = true
	return nil
}

type Payment struct{ attempts int64 }

// Charge captures payment for an order. It fails the first time it is
// called (a transient outage) and succeeds afterwards — so the checkout
// flow crashes mid-run the first time and recovers on resume.
// @example {"order": "order-1"}
func (s *Payment) Charge(_ context.Context, req *Order, rsp *Order) error {
	n := atomic.AddInt64(&s.attempts, 1)
	if n == 1 {
		fmt.Printf("    \033[31m[payment]\033[0m gateway timeout (attempt %d)\n", n)
		return fmt.Errorf("payment gateway timeout")
	}
	fmt.Printf("    \033[32m[payment]\033[0m charged %s (attempt %d)\n", req.Order, n)
	*rsp = *req
	rsp.Charged = true
	return nil
}

type Orders struct{ confirms int64 }

// Confirm finalizes an order.
// @example {"order": "order-1"}
func (s *Orders) Confirm(_ context.Context, req *Order, rsp *Order) error {
	atomic.AddInt64(&s.confirms, 1)
	fmt.Printf("    \033[32m[orders]\033[0m confirmed %s\n", req.Order)
	*rsp = *req
	rsp.Confirmed = true
	return nil
}

type SendRequest struct {
	To      string `json:"to" description:"Recipient (required)"`
	Message string `json:"message" description:"Message body (required)"`
}
type SendResponse struct {
	Sent bool `json:"sent"`
}

type Notify struct{ sent int64 }

// Send delivers a notification.
// @example {"to": "buyer@acme.com", "message": "Your order is confirmed"}
func (s *Notify) Send(_ context.Context, req *SendRequest, rsp *SendResponse) error {
	atomic.AddInt64(&s.sent, 1)
	fmt.Printf("    \033[35m[notify]\033[0m 📨 to=%s %q\n", req.To, req.Message)
	rsp.Sent = true
	return nil
}

// ---------------------------------------------------------------------------
// mock LLM — the only fake. The concierge agent uses it to decide to notify.
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

func (m *mockModel) Generate(ctx context.Context, req *ai.Request, _ ...ai.GenerateOption) (*ai.Response, error) {
	// The concierge is asked to notify the buyer. Find the notify tool and call it.
	for _, t := range req.Tools {
		if strings.Contains(t.Name, "Send") && m.opts.ToolHandler != nil {
			m.opts.ToolHandler(ctx, ai.ToolCall{
				ID:    "call-1",
				Name:  t.Name,
				Input: map[string]any{"to": "buyer@acme.com", "message": "Your order is confirmed."},
			})
			break
		}
	}
	return &ai.Response{Answer: "Buyer notified."}, nil
}

// ---------------------------------------------------------------------------
// assertions
// ---------------------------------------------------------------------------

var failures int

func check(cond bool, format string, args ...any) {
	if cond {
		fmt.Printf("  \033[32m✓\033[0m %s\n", fmt.Sprintf(format, args...))
		return
	}
	fmt.Printf("  \033[31m✗ %s\033[0m\n", fmt.Sprintf(format, args...))
	failures++
}

// a2aReachable calls the named agent through the gateway using the A2A
// client — exercising both directions of the protocol — and reports
// whether the agent replied.
func a2aReachable(base, agent string) bool {
	reply, err := a2a.NewClient(base+"/agents/"+agent).Send(context.Background(), "notify the buyer")
	return err == nil && reply != ""
}

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

func main() {
	provider := flag.String("provider", "mock", "LLM provider: mock (default), anthropic, openai, ...")
	flag.Parse()
	os.Exit(runUniverse(*provider))
}

func runUniverse(provider string) int {
	failures = 0
	apiKey := ""
	if provider == "mock" {
		ai.Register("mock", newMock)
	} else if apiKey = providerKey(provider); apiKey == "" {
		fmt.Printf("no API key for provider %q — set MICRO_AI_API_KEY or the provider's key env\n", provider)
		return 2
	}

	fmt.Printf("\n\033[1mUNIVERSE — booting a mini go-micro world (provider: %s)\033[0m\n\n", provider)

	// Infrastructure — all in-memory, all real.
	reg := registry.NewMemoryRegistry()
	br := broker.NewMemoryBroker()
	if err := br.Connect(); err != nil {
		fmt.Println("broker connect:", err)
		return 2
	}
	cl := harnessutil.Client(provider, reg)
	st := store.NewMemoryStore()
	liveAgentOpts := harnessutil.AgentOptions(provider)

	// Services.
	inv, pay, ord, ntf := new(Inventory), new(Payment), new(Orders), new(Notify)
	for name, h := range map[string]any{"inventory": inv, "payment": pay, "orders": ord, "notify": ntf} {
		svc := service.New(service.Name(name), service.Address("127.0.0.1:0"), service.Registry(reg), service.Client(cl))
		svc.Handle(h)
		go svc.Run()
	}

	// The concierge agent: guardrails on, plus a tool-execution wrapper
	// that counts calls — to prove the wrapper seam runs end-to-end.
	var wrapped int64
	conciergeOpts := []agent.Option{
		agent.Name("concierge"),
		agent.Services("notify"),
		agent.Prompt("You notify buyers when their order is confirmed."),
		agent.Address("127.0.0.1:0"),
		agent.Provider(provider), agent.APIKey(apiKey),
		agent.MaxSteps(5),
		agent.WrapTool(func(next ai.ToolHandler) ai.ToolHandler {
			return func(ctx context.Context, call ai.ToolCall) ai.ToolResult {
				atomic.AddInt64(&wrapped, 1)
				return next(ctx, call)
			}
		}),
		agent.WithRegistry(reg), agent.WithClient(cl), agent.WithStore(st),
	}
	conciergeOpts = append(conciergeOpts, liveAgentOpts...)
	concierge := agent.New(conciergeOpts...)
	go concierge.Run()
	defer concierge.Stop()

	waitFor(reg, "inventory", "payment", "orders", "notify", "concierge")

	// The durable checkout flow: ordered, checkpointed steps. The last step
	// hands off to the concierge agent.
	checkout := flow.New("checkout",
		flow.Trigger("events.order.placed"),
		flow.WithCheckpoint(flow.StoreCheckpoint(st, "checkout")),
		flow.Timeout(harnessutil.LiveTimeout(provider)),
		flow.Steps(
			flow.Step{Name: "reserve", Run: flow.Call("inventory", "Inventory.Reserve")},
			flow.Step{Name: "charge", Run: flow.Call("payment", "Payment.Charge")},
			flow.Step{Name: "confirm", Run: flow.Call("orders", "Orders.Confirm")},
			flow.Step{Name: "notify", Run: flow.Dispatch("concierge")},
		),
	)
	if err := checkout.Register(reg, br, cl); err != nil {
		fmt.Println("flow register:", err)
		return 2
	}
	defer checkout.Stop()

	ctx := context.Background()

	// 1) An order event triggers the flow — which crashes at "charge".
	fmt.Println("\033[1m> event:\033[0m events.order.placed {\"order\":\"order-1\"}")
	if err := br.Publish("events.order.placed", &broker.Message{Body: []byte(`{"order":"order-1"}`)}); err != nil {
		fmt.Println("publish:", err)
		return 2
	}

	// Wait for the run to be checkpointed as failed at "charge".
	var runID string
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if pend, _ := checkout.Pending(ctx); len(pend) == 1 && pend[0].State.Stage == "charge" {
			runID = pend[0].ID
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	fmt.Println("\n\033[1mafter crash:\033[0m")
	check(runID != "", "flow checkpointed a pending run at the failing step")
	check(atomic.LoadInt64(&inv.reserves) == 1, "inventory reserved once before the crash")
	check(atomic.LoadInt64(&ord.confirms) == 0, "order not confirmed yet (run stopped at charge)")

	// 2) Resume — the dependency has recovered; continue from "charge".
	fmt.Println("\n\033[1m> resume:\033[0m", runID)
	if runID != "" {
		if err := checkout.Resume(ctx, runID); err != nil {
			fmt.Println("resume:", err)
		}
	}

	// Wait for the agent (last step) to have notified.
	deadline = time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if atomic.LoadInt64(&ntf.sent) >= 1 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	// 3) Assert the end state of the universe.
	fmt.Println("\n\033[1mafter resume:\033[0m")
	check(atomic.LoadInt64(&inv.reserves) == 1, "inventory still reserved exactly once (completed step not replayed)")
	check(atomic.LoadInt64(&pay.attempts) == 2, "payment attempted twice (failed once, then charged)")
	check(atomic.LoadInt64(&ord.confirms) == 1, "order confirmed after resume")
	check(atomic.LoadInt64(&ntf.sent) >= 1, "buyer notified by the concierge agent")
	check(atomic.LoadInt64(&wrapped) >= 1, "agent tool-execution wrapper observed the call")

	if pend, _ := checkout.Pending(ctx); true {
		check(len(pend) == 0, "no pending runs — the workflow completed durably")
	}

	// Scoped state landed in its own tables.
	runs, _ := flow.StoreCheckpoint(st, "checkout").List(ctx)
	check(len(runs) == 1 && runs[0].Status == "done", "flow run persisted in flow/checkout and marked done")
	hist := agent.NewMemory(store.Scope(st, "agent", "concierge"), "history", 100).Messages()
	check(len(hist) > 0, "agent history persisted in agent/concierge")

	// Flows are discoverable while live.
	flows, _ := reg.GetService("checkout")
	check(len(flows) == 1 && flows[0].Metadata["type"] == "flow", "flow registered in the registry as type=flow")

	// 4) The concierge agent is also reachable from outside, over the A2A
	// protocol — its card is generated from the registry, and a task is
	// translated to its Agent.Chat RPC.
	gw := httptest.NewServer(a2a.New(a2a.Options{Registry: reg, Client: cl, BaseURL: "http://gw"}).Handler())
	defer gw.Close()
	check(a2aReachable(gw.URL, "concierge"), "concierge agent reachable over the A2A gateway")

	fmt.Println("\n\033[1m> shutting down the universe\033[0m")
	// defers stop the agent and flow (deregistering them).

	if failures > 0 {
		fmt.Printf("\n\033[31m✗ universe failed: %d assertion(s)\033[0m\n", failures)
		return 1
	}
	fmt.Println("\n\033[32m✓ universe: booted, survived a crash, resumed, and shut down cleanly\033[0m")
	return 0
}
