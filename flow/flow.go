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
	"strconv"
	"sync"
	"text/template"
	"time"

	"github.com/google/uuid"
	"go-micro.dev/v6/ai"
	"go-micro.dev/v6/broker"
	"go-micro.dev/v6/client"
	codecbytes "go-micro.dev/v6/codec/bytes"
	"go-micro.dev/v6/logger"
	"go-micro.dev/v6/registry"

	// Register default providers.
	_ "go-micro.dev/v6/ai/anthropic"
	_ "go-micro.dev/v6/ai/atlascloud"
	_ "go-micro.dev/v6/ai/gemini"
	_ "go-micro.dev/v6/ai/groq"
	_ "go-micro.dev/v6/ai/mistral"
	_ "go-micro.dev/v6/ai/openai"
	_ "go-micro.dev/v6/ai/together"
)

// Flow is an event-driven LLM orchestration unit. It subscribes to
// a broker topic, discovers services as tools, and feeds each event
// into an LLM that decides which RPCs to call.
type Flow struct {
	name         string
	opts         Options
	model        ai.Model
	toolSet      *ai.Tools
	client       client.Client
	tmpl         *template.Template
	log          logger.Logger
	checkpoint   Checkpoint
	reg          registry.Registry
	sub          broker.Subscriber
	registration *registry.Service
	mu           sync.Mutex
	results      []Result
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
		name:       name,
		opts:       o,
		tmpl:       tmpl,
		log:        logger.DefaultLogger,
		checkpoint: defaultCheckpoint(name, o),
	}
}

// Register wires the flow into a running service. It sets up the
// model, discovers tools from the registry, and subscribes to the
// trigger topic on the broker. Call this before service.Run().
func (f *Flow) Register(reg registry.Registry, br broker.Broker, cl client.Client) error {
	f.client = cl
	f.reg = reg
	f.toolSet = ai.NewTools(reg, ai.ToolClient(cl))

	// A flow that dispatches to an agent doesn't run its own model — the
	// agent is the engine. Otherwise, set up the augmented LLM.
	if f.opts.Agent == "" {
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
	}

	if f.opts.TriggerTopic != "" {
		sub, err := br.Subscribe(f.opts.TriggerTopic, func(p broker.Event) error {
			data := string(p.Message().Body)
			if err := f.Execute(context.Background(), data); err != nil {
				f.log.Logf(logger.ErrorLevel, "Flow %s failed: %v", f.name, err)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("subscribe to %s: %w", f.opts.TriggerTopic, err)
		}
		f.sub = sub
		f.log.Logf(logger.InfoLevel, "Flow %s subscribed to %s", f.name, f.opts.TriggerTopic)

		// Announce the flow in the registry so it's discoverable like a
		// service or agent (e.g. `micro flow list`). This is liveness only:
		// Stop deregisters it. Durable run history lives in the store.
		f.registration = &registry.Service{
			Name:    f.name,
			Version: "latest",
			Metadata: map[string]string{
				"type":    "flow",
				"trigger": f.opts.TriggerTopic,
				"steps":   strconv.Itoa(len(f.opts.Steps)),
			},
			Nodes: []*registry.Node{{
				Id:       f.name + "-" + uuid.New().String()[:8],
				Address:  "flow://" + f.name,
				Metadata: map[string]string{"type": "flow"},
			}},
		}
		if err := reg.Register(f.registration); err != nil {
			f.log.Logf(logger.ErrorLevel, "Flow %s registry register: %v", f.name, err)
			f.registration = nil
		}
	}

	return nil
}

// Stop unsubscribes the flow from its trigger and deregisters it from the
// registry. In-flight and past runs remain in the store; Stop only ends
// the flow's liveness, mirroring how a service leaves the registry when
// it shuts down.
func (f *Flow) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	if f.opts.Timeout <= 0 {
		return ctx, func() {}
	}
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, f.opts.Timeout)
}

func (f *Flow) Stop() error {
	if f.sub != nil {
		_ = f.sub.Unsubscribe()
		f.sub = nil
	}
	if f.registration != nil && f.reg != nil {
		err := f.reg.Deregister(f.registration)
		f.registration = nil
		return err
	}
	return nil
}

// Execute runs the flow once with the given input data. This is
// called automatically on each broker event, but can also be
// invoked directly for testing or one-shot use.
func (f *Flow) Execute(ctx context.Context, data string) error {
	ctx, cancel := f.withTimeout(ctx)
	defer cancel()

	// Stepped flows run the ordered, checkpointed step loop.
	if len(f.opts.Steps) > 0 {
		_, err := f.startRun(ctx, data)
		return err
	}

	runID := uuid.New().String()
	ctx = ai.WithRunInfo(ctx, ai.RunInfo{RunID: runID, Flow: f.name})

	start := time.Now()

	prompt := data
	if f.tmpl != nil {
		var buf bytes.Buffer
		_ = f.tmpl.Execute(&buf, map[string]string{"Data": data})
		prompt = buf.String()
	}

	result := Result{
		FlowName:  f.name,
		Trigger:   f.opts.TriggerTopic,
		Prompt:    prompt,
		Timestamp: start,
	}

	// Flow triggers, Agent reasons: hand the event to the named agent.
	if f.opts.Agent != "" {
		reply, err := f.callAgent(ctx, f.opts.Agent, prompt)
		result.Duration = time.Since(start).Seconds()
		if err != nil {
			result.Error = err.Error()
			f.record(result)
			return err
		}
		result.Reply = reply
		f.record(result)
		f.log.Logf(logger.InfoLevel, "Flow %s dispatched to agent %s in %.1fs",
			f.name, f.opts.Agent, result.Duration)
		return nil
	}

	// Otherwise run a single augmented-LLM step with the services as tools.
	discovered, err := f.toolSet.Discover()
	if err != nil {
		result.Duration = time.Since(start).Seconds()
		result.Error = err.Error()
		f.record(result)
		return fmt.Errorf("discover tools: %w", err)
	}

	resp, err := f.model.Generate(ctx, &ai.Request{
		Prompt:       prompt,
		SystemPrompt: f.opts.SystemPrompt,
		Tools:        discovered,
	})
	result.Duration = time.Since(start).Seconds()

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

// callAgent hands the rendered prompt to a registered agent's Agent.Chat
// endpoint over RPC and returns its reply.
func (f *Flow) callAgent(ctx context.Context, name, message string) (string, error) {
	info, _ := ai.RunInfoFrom(ctx)
	body, _ := json.Marshal(map[string]string{"message": message, "parent_id": info.RunID})
	req := f.client.NewRequest(name, "Agent.Chat", &codecbytes.Frame{Data: body})
	var rsp codecbytes.Frame
	if err := f.client.Call(ctx, req, &rsp); err != nil {
		return "", err
	}
	var out struct {
		Reply string `json:"reply"`
	}
	if err := json.Unmarshal(rsp.Data, &out); err != nil {
		return "", err
	}
	return out.Reply, nil
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
