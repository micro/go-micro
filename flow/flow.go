// Package flow provides event-driven workflows for go-micro services.
//
// A Flow is a workflow in the sense of Anthropic's "Building Effective
// Agents": LLMs and tools orchestrated through a predefined path. It
// subscribes to a broker topic and, for each event, runs one augmented
// LLM step — the registered services as tools, a fixed prompt — and
// lets the model decide which RPCs to call. Use a Flow when the task is
// well-defined and you want a deterministic trigger; use an Agent (see
// the agent package) when the work needs to direct itself dynamically.
//
// Usage:
//
//	f := flow.New("onboard-user",
//	    flow.Trigger("events.user.created"),
//	    flow.Prompt("New user created: {{.Data}}. Send welcome email and create workspace."),
//	    flow.Provider("anthropic"),
//	    flow.APIKey(key),
//	)
//	f.Register(service)
//	service.Run()
package flow

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"text/template"
	"time"

	"go-micro.dev/v5/ai"
	"go-micro.dev/v5/broker"
	"go-micro.dev/v5/client"
	"go-micro.dev/v5/logger"
	"go-micro.dev/v5/registry"

	// Register default providers.
	_ "go-micro.dev/v5/ai/anthropic"
	_ "go-micro.dev/v5/ai/atlascloud"
	_ "go-micro.dev/v5/ai/gemini"
	_ "go-micro.dev/v5/ai/groq"
	_ "go-micro.dev/v5/ai/mistral"
	_ "go-micro.dev/v5/ai/openai"
	_ "go-micro.dev/v5/ai/together"
)

// Flow is an event-driven LLM orchestration unit. It subscribes to
// a broker topic, discovers services as tools, and feeds each event
// into an LLM that decides which RPCs to call.
type Flow struct {
	name    string
	opts    Options
	model   ai.Model
	toolSet *ai.Tools
	tmpl    *template.Template
	log     logger.Logger
	mu      sync.Mutex
	results []Result
}

// Result records one flow execution.
type Result struct {
	FlowName  string    `json:"flow"`
	Trigger   string    `json:"trigger"`
	Prompt    string    `json:"prompt"`
	Reply     string    `json:"reply,omitempty"`
	Answer    string    `json:"answer,omitempty"`
	ToolCalls []string  `json:"tool_calls,omitempty"`
	Error     string    `json:"error,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	Duration  float64   `json:"duration_seconds"`
}

// New creates a Flow with the given name and options.
func New(name string, opts ...Option) *Flow {
	o := Options{
		Provider:     "openai",
		SystemPrompt: "You are a service orchestrator. Use the available tools to fulfill the request. Explain what you do.",
		HistoryLimit: 20,
	}
	for _, opt := range opts {
		opt(&o)
	}

	var tmpl *template.Template
	if o.Prompt != "" {
		var err error
		tmpl, err = template.New(name).Parse(o.Prompt)
		if err != nil {
			tmpl = template.Must(template.New(name).Parse("{{.Data}}"))
		}
	}

	return &Flow{
		name: name,
		opts: o,
		tmpl: tmpl,
		log:  logger.DefaultLogger,
	}
}

// Register wires the flow into a running service. It sets up the
// model, discovers tools from the registry, and subscribes to the
// trigger topic on the broker. Call this before service.Run().
func (f *Flow) Register(reg registry.Registry, br broker.Broker, cl client.Client) error {
	f.toolSet = ai.NewTools(reg, ai.ToolClient(cl))

	var modelOpts []ai.Option
	if f.opts.APIKey != "" {
		modelOpts = append(modelOpts, ai.WithAPIKey(f.opts.APIKey))
	}
	if f.opts.Model != "" {
		modelOpts = append(modelOpts, ai.WithModel(f.opts.Model))
	}
	if f.opts.BaseURL != "" {
		modelOpts = append(modelOpts, ai.WithBaseURL(f.opts.BaseURL))
	}
	modelOpts = append(modelOpts, ai.WithTools(f.toolSet))

	f.model = ai.New(f.opts.Provider, modelOpts...)
	if f.model == nil {
		return fmt.Errorf("unknown provider: %s", f.opts.Provider)
	}

	if f.opts.TriggerTopic != "" {
		_, err := br.Subscribe(f.opts.TriggerTopic, func(p broker.Event) error {
			data := string(p.Message().Body)
			if err := f.Execute(context.Background(), data); err != nil {
				f.log.Logf(logger.ErrorLevel, "Flow %s failed: %v", f.name, err)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("subscribe to %s: %w", f.opts.TriggerTopic, err)
		}
		f.log.Logf(logger.InfoLevel, "Flow %s subscribed to %s", f.name, f.opts.TriggerTopic)
	}

	return nil
}

// Execute runs the flow once with the given input data. This is
// called automatically on each broker event, but can also be
// invoked directly for testing or one-shot use.
func (f *Flow) Execute(ctx context.Context, data string) error {
	start := time.Now()

	discovered, err := f.toolSet.Discover()
	if err != nil {
		return fmt.Errorf("discover tools: %w", err)
	}

	prompt := data
	if f.tmpl != nil {
		var buf bytes.Buffer
		f.tmpl.Execute(&buf, map[string]string{"Data": data})
		prompt = buf.String()
	}

	resp, err := f.model.Generate(ctx, &ai.Request{
		Prompt:       prompt,
		SystemPrompt: f.opts.SystemPrompt,
		Tools:        discovered,
	})

	result := Result{
		FlowName:  f.name,
		Trigger:   f.opts.TriggerTopic,
		Prompt:    prompt,
		Timestamp: start,
		Duration:  time.Since(start).Seconds(),
	}

	if err != nil {
		result.Error = err.Error()
		f.record(result)
		return err
	}

	result.Reply = resp.Reply
	result.Answer = resp.Answer
	for _, tc := range resp.ToolCalls {
		args, _ := json.Marshal(tc.Input)
		result.ToolCalls = append(result.ToolCalls, fmt.Sprintf("%s(%s)", tc.Name, args))
	}

	f.record(result)

	f.log.Logf(logger.InfoLevel, "Flow %s completed in %.1fs: %d tool calls",
		f.name, result.Duration, len(result.ToolCalls))

	return nil
}

// Results returns a copy of all recorded execution results.
func (f *Flow) Results() []Result {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]Result, len(f.results))
	copy(out, f.results)
	return out
}

// Name returns the flow name.
func (f *Flow) Name() string {
	return f.name
}

func (f *Flow) record(r Result) {
	f.mu.Lock()
	f.results = append(f.results, r)
	f.mu.Unlock()

	if f.opts.OnResult != nil {
		f.opts.OnResult(r)
	}
}
