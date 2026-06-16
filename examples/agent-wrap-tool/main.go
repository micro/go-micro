// Agent Tool Wrappers — middleware around tool execution
//
// Every tool call an agent makes runs through ai.ToolHandler. WrapTool
// wraps that handler the same way client.CallWrapper and
// server.HandlerWrapper wrap RPCs: a wrapper takes the next handler and
// returns a new one, so code before the next(...) call runs before the
// tool and code after runs after.
//
// This example registers two wrappers on one agent:
//
//   - observe: times every call and records a count per tool, keyed so
//     you can correlate by call ID. Pure "lifecycle hook" — it observes,
//     it doesn't change behaviour.
//   - retry:   re-runs a call whose result comes back as an error, up to
//     a few attempts. The "weather" service fails the first time it is
//     hit and succeeds after, so the retry wrapper turns a transient
//     failure into a success without the model ever seeing it.
//
// Wrappers compose outermost-first, so observe (registered first) wraps
// retry: it sees one logical call even though retry may run it twice.
//
// Run (needs an LLM provider key):
//
//	MICRO_AI_PROVIDER=anthropic MICRO_AI_API_KEY=sk-ant-... go run main.go
package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"go-micro.dev/v5"
	"go-micro.dev/v5/ai"
)

// ---------------------------------------------------------------------------
// weather service — flaky on purpose
// ---------------------------------------------------------------------------

type ForecastRequest struct {
	City string `json:"city" description:"City to get the forecast for (required)"`
}

type ForecastResponse struct {
	City     string `json:"city" description:"The city"`
	Forecast string `json:"forecast" description:"Human-readable forecast"`
}

type WeatherService struct {
	mu    sync.Mutex
	calls int
}

// Forecast returns the weather forecast for a city. It fails on the very
// first call (to simulate a transient dependency error) and succeeds
// afterwards, so the retry wrapper has something to recover from.
//
// @example {"city": "London"}
func (s *WeatherService) Forecast(ctx context.Context, req *ForecastRequest, rsp *ForecastResponse) error {
	s.mu.Lock()
	s.calls++
	n := s.calls
	s.mu.Unlock()

	if n == 1 {
		return fmt.Errorf("weather upstream temporarily unavailable")
	}
	rsp.City = req.City
	rsp.Forecast = "18°C and clear"
	return nil
}

// ---------------------------------------------------------------------------
// wrappers
// ---------------------------------------------------------------------------

// metrics is collected by the observe wrapper. A real deployment would
// emit these to OpenTelemetry or Prometheus; here we just print them.
type metrics struct {
	mu     sync.Mutex
	counts map[string]int
	took   map[string]time.Duration
}

func newMetrics() *metrics {
	return &metrics{counts: map[string]int{}, took: map[string]time.Duration{}}
}

// observe times each tool call and records a per-tool count. It mirrors a
// service-side metrics wrapper: measure around next(...), record, return
// the result untouched.
func (m *metrics) observe(next ai.ToolHandler) ai.ToolHandler {
	return func(ctx context.Context, call ai.ToolCall) ai.ToolResult {
		start := time.Now()
		res := next(ctx, call)
		took := time.Since(start)

		m.mu.Lock()
		m.counts[call.Name]++
		m.took[call.Name] += took
		m.mu.Unlock()

		fmt.Printf("  [observe] id=%s tool=%s took=%s\n", shortID(call.ID), call.Name, took.Round(time.Millisecond))
		return res
	}
}

// retry re-runs a call whose result comes back as an error, up to
// attempts times. Because it sits inside observe, the outer wrapper still
// sees one logical call even though retry may run next more than once.
//
// Developer wrappers run outside the built-in guardrails, so next here is
// the full guardrail stack: each retry is also seen by loop detection.
// Keep LoopLimit at or above your retry count (the default 3 covers the
// 2 attempts this example makes), or disable it with AgentLoopLimit(0)
// when a wrapper is responsible for repetition.
func retry(attempts int) ai.ToolWrapper {
	return func(next ai.ToolHandler) ai.ToolHandler {
		return func(ctx context.Context, call ai.ToolCall) ai.ToolResult {
			var res ai.ToolResult
			for i := 1; i <= attempts; i++ {
				res = next(ctx, call)
				if !isError(res) {
					return res
				}
				if i < attempts {
					fmt.Printf("  [retry]   tool=%s attempt %d failed, retrying\n", call.Name, i)
				}
			}
			return res
		}
	}
}

// isError reports whether a tool result is an error. The RPC handler
// encodes failures as a JSON object with an "error" field in Content.
func isError(res ai.ToolResult) bool {
	return strings.Contains(res.Content, `"error"`)
}

func shortID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	if id == "" {
		return "-"
	}
	return id
}

func main() {
	provider, apiKey := detectProvider()
	if apiKey == "" {
		fmt.Println("No LLM key found. Set a provider key and run again, e.g.:")
		fmt.Println("  export ANTHROPIC_API_KEY=sk-ant-...   # or OPENAI_API_KEY, GEMINI_API_KEY, ...")
		fmt.Println("  go run main.go")
		return
	}
	fmt.Printf("Using provider %q\n", provider)

	weather := micro.New("weather")
	weather.Handle(new(WeatherService))
	go weather.Run()

	m := newMetrics()

	agent := micro.NewAgent("forecaster",
		micro.AgentServices("weather"),
		micro.AgentPrompt("You report the weather. Use the weather service to answer."),
		micro.AgentProvider(provider),
		micro.AgentAPIKey(apiKey),
		// observe is registered first, so it is the outer wrapper and
		// retry is the inner one.
		micro.AgentWrapTool(m.observe, retry(3)),
	)

	// Give the service a moment to register.
	time.Sleep(2 * time.Second)

	resp, err := agent.Ask(context.Background(), "What's the weather in London?")
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}

	fmt.Println("\n--- reply ---")
	fmt.Println(resp.Reply)

	fmt.Println("\n--- tool metrics ---")
	m.mu.Lock()
	for name, n := range m.counts {
		fmt.Printf("  %s: %d call(s), total %s\n", name, n, m.took[name].Round(time.Millisecond))
	}
	m.mu.Unlock()
}

// detectProvider picks an LLM provider and key from the environment.
// MICRO_AI_PROVIDER / MICRO_AI_API_KEY win if set; otherwise it falls
// back to the first provider-specific key it finds.
func detectProvider() (provider, apiKey string) {
	provider = os.Getenv("MICRO_AI_PROVIDER")
	apiKey = os.Getenv("MICRO_AI_API_KEY")
	if apiKey != "" {
		if provider == "" {
			provider = "anthropic"
		}
		return provider, apiKey
	}

	for _, p := range []struct{ name, env string }{
		{"anthropic", "ANTHROPIC_API_KEY"},
		{"openai", "OPENAI_API_KEY"},
		{"gemini", "GEMINI_API_KEY"},
		{"groq", "GROQ_API_KEY"},
		{"mistral", "MISTRAL_API_KEY"},
		{"together", "TOGETHER_API_KEY"},
		{"atlascloud", "ATLASCLOUD_API_KEY"},
	} {
		if v := os.Getenv(p.env); v != "" {
			return p.name, v
		}
	}
	return "", ""
}
