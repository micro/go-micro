package harnessutil

import (
	"fmt"
	"os"
	"time"

	"go-micro.dev/v6/agent"
	"go-micro.dev/v6/client"
	"go-micro.dev/v6/registry"
	"go-micro.dev/v6/selector"
)

const (
	// LiveTimeoutEnv overrides the per-call deadline used by live-provider
	// harness runs. It intentionally does not affect deterministic mock runs.
	LiveTimeoutEnv = "GO_MICRO_HARNESS_LIVE_TIMEOUT"
	// DefaultLiveTimeout is generous enough for slow but correct hosted models
	// while still bounding genuinely stuck live conformance runs.
	DefaultLiveTimeout = 5 * time.Minute
)

// LiveTimeout returns the harness per-call timeout for live providers. Mock runs
// keep their historical fast defaults by returning zero.
func LiveTimeout(provider string) time.Duration {
	if provider == "mock" {
		return 0
	}
	if raw := os.Getenv(LiveTimeoutEnv); raw != "" {
		d, err := time.ParseDuration(raw)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid %s=%q; using %s\n", LiveTimeoutEnv, raw, DefaultLiveTimeout)
			return DefaultLiveTimeout
		}
		return d
	}
	return DefaultLiveTimeout
}

// Client returns an in-memory-registry client. Live provider harnesses get a
// larger request timeout so an otherwise correct agent run is not cut off by the
// default 30-second RPC deadline; mock runs are unchanged.
func Client(provider string, reg registry.Registry) client.Client {
	opts := []client.Option{
		client.Registry(reg),
		client.Selector(selector.NewSelector(selector.Registry(reg))),
	}
	if d := LiveTimeout(provider); d > 0 {
		opts = append(opts, client.RequestTimeout(d))
	}
	return client.NewClient(opts...)
}

// AgentOptions applies the same live-provider timeout to model and tool calls.
// The empty result for mock runs preserves their deterministic timing.
func AgentOptions(provider string) []agent.Option {
	if d := LiveTimeout(provider); d > 0 {
		return []agent.Option{agent.ModelCallTimeout(d), agent.ToolCallTimeout(d)}
	}
	return nil
}
