// Agent x402 buyer — a provider-free example of an agent paying for a paid tool.
//
// Run:
//
//	go run ./examples/agent-x402-buyer
//
// It starts a local HTTP tool protected by x402 middleware, then asks a
// deterministic mock-model agent to call that tool. The agent receives the 402
// challenge, uses AgentPayer and AgentBudget to pay within a local mock
// facilitator, retries the request, and prints the run spend.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"

	go_micro "go-micro.dev/v6"
	"go-micro.dev/v6/agent"
	"go-micro.dev/v6/ai"
	"go-micro.dev/v6/store"
	"go-micro.dev/v6/wrapper/x402"
)

const (
	paidToolName = "paid.market_brief"
	price        = int64(7)
	paymentToken = "dev-payment-token"
)

type devFacilitator struct {
	verifyCount int
	settleCount int
}

func (f *devFacilitator) Verify(ctx context.Context, payment string, req x402.Requirements) (x402.Result, error) {
	f.verifyCount++
	if payment != paymentToken {
		return x402.Result{Valid: false, Reason: "unknown dev payment token"}, nil
	}
	return x402.Result{Valid: true, Payer: "dev-agent-wallet"}, nil
}

func (f *devFacilitator) Settle(ctx context.Context, payment string, req x402.Requirements) (x402.Result, error) {
	f.settleCount++
	return x402.Result{Valid: true, Settlement: "dev-settlement-001"}, nil
}

type devPayer struct{}

func (devPayer) Pay(ctx context.Context, req x402.Requirements) (string, error) {
	return paymentToken, nil
}

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
func (m *mockModel) String() string      { return "agent-x402-buyer-mock" }
func (m *mockModel) Stream(context.Context, *ai.Request, ...ai.GenerateOption) (ai.Stream, error) {
	return nil, fmt.Errorf("stream not supported by agent-x402-buyer mock")
}

func (m *mockModel) Generate(ctx context.Context, req *ai.Request, _ ...ai.GenerateOption) (*ai.Response, error) {
	for _, tool := range req.Tools {
		if tool.Name == paidToolName && m.opts.ToolHandler != nil {
			out := m.opts.ToolHandler(ctx, ai.ToolCall{ID: "paid-brief", Name: tool.Name, Input: map[string]any{"url": req.Prompt}})
			return &ai.Response{Answer: fmt.Sprintf("Paid tool returned: %s", out.Content)}, nil
		}
	}
	return &ai.Response{Answer: "No paid tool was available."}, nil
}

func paidToolServer(fac *devFacilitator) *httptest.Server {
	mux := http.NewServeMux()
	paid := x402.Middleware(x402.Config{
		PayTo:       "0xMerchantDevWallet",
		Network:     "base-sepolia",
		Amount:      fmt.Sprint(price),
		Description: "Local market brief for the x402 buyer example",
		Facilitator: fac,
	})
	mux.Handle("/brief", paid(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"brief":      "Mock demand is up 12% after the agent paid the local tool.",
			"settlement": w.Header().Get(x402.PaymentResponseHeader),
		})
	})))
	return httptest.NewServer(mux)
}

func run(w io.Writer) error {
	ai.Register("agent-x402-buyer-mock", newMock)

	fac := &devFacilitator{}
	srv := paidToolServer(fac)
	defer srv.Close()

	st := store.NewMemoryStore()
	buyer := agent.New(
		agent.Name("x402-buyer"),
		agent.Provider("agent-x402-buyer-mock"),
		agent.Prompt("Call the paid market brief tool when given its URL."),
		agent.WithStore(st),
		go_micro.AgentPayer(devPayer{}),
		go_micro.AgentBudget(10),
		agent.WithTool(paidToolName, "Fetch a paid market brief over HTTP", map[string]any{
			"url": map[string]any{"type": "string", "description": "Paid HTTP endpoint to call"},
		}, func(ctx context.Context, input map[string]any) (string, error) {
			url, _ := input["url"].(string)
			resp, err := http.Get(url)
			if err != nil {
				return "", err
			}
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return "", err
			}
			return string(body), nil
		}),
	)

	resp, err := buyer.Ask(context.Background(), srv.URL+"/brief")
	if err != nil {
		return err
	}
	events, err := agent.LoadRunEvents(st, "x402-buyer", resp.RunID)
	if err != nil {
		return err
	}
	var spent int64
	for _, event := range events {
		if event.Spent > spent {
			spent = event.Spent
		}
	}

	fmt.Fprintln(w, "Agent x402 buyer (provider: mock, funds: local dev token)")
	fmt.Fprintln(w, strings.TrimSpace(resp.Reply))
	fmt.Fprintf(w, "facilitator verify=%d settle=%d\n", fac.verifyCount, fac.settleCount)
	fmt.Fprintf(w, "run spend: %d smallest units (budget 10)\n", spent)
	return nil
}

func main() {
	if err := run(os.Stdout); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
