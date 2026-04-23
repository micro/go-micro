// Package agent provides AI agents that manage the lifecycle of services.
// Agents use tools to observe and control services, driven by a directive
// that describes their purpose. They operate externally to the services
// they manage, interacting through the registry and RPC client.
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"go-micro.dev/v5/codec/bytes"
	log "go-micro.dev/v5/logger"
	"go-micro.dev/v5/model"
	"go-micro.dev/v5/registry"
)

// ActivityType classifies the kind of action an agent has performed.
type ActivityType string

const (
	// ActivityEvaluate marks a periodic evaluation cycle.
	ActivityEvaluate ActivityType = "evaluate"
	// ActivityPrompt marks an on-demand prompt submitted via Prompt.
	ActivityPrompt ActivityType = "prompt"
	// ActivityTool marks a tool invocation made by the model.
	ActivityTool ActivityType = "tool"
	// ActivityResponse marks a completed model response.
	ActivityResponse ActivityType = "response"
	// ActivityError marks an error that occurred during processing.
	ActivityError ActivityType = "error"
)

// Activity records a single action performed by the agent.
type Activity struct {
	// Time is when the activity occurred.
	Time time.Time
	// Type classifies the activity.
	Type ActivityType
	// Prompt is the text of the prompt that triggered the activity (if any).
	Prompt string
	// Tool is the name of the tool invoked (for ActivityTool).
	Tool string
	// Result holds the output of a tool call or model response.
	Result string
	// Err holds any error that occurred (for ActivityError).
	Err error
}

// maxActivities is the maximum number of Activity entries kept in memory.
const maxActivities = 256

// Agent manages the lifecycle of services using AI-driven tools.
// Its interface mirrors the Service interface so agents can live alongside
// services in the same runtime environment.
type Agent interface {
	// Init initializes the agent with options
	Init(...Option) error
	// Options returns the current options
	Options() Options
	// Run starts the agent loop
	Run() error
	// Stop gracefully stops the agent
	Stop() error
	// String returns the agent name
	String() string
	// Prompt queues a user-provided prompt for the agent to process immediately.
	// The call is non-blocking and returns a channel that will receive the model
	// response once the prompt has been evaluated (and any requested tools have
	// been executed). The channel is buffered and closed after the response is
	// sent, so callers can range over it or select on it.
	Prompt(text string) <-chan *model.Response
	// Activity returns a chronological snapshot of recent agent activities
	// (evaluations, prompts, tool calls, responses, and errors).
	Activity() []Activity
}

// agent is the default Agent implementation.
type agent struct {
	opts       Options
	stop       chan struct{}
	once       sync.Once
	activities []Activity
	actMu      sync.RWMutex
}

// New creates a new Agent with the given options.
func New(opts ...Option) Agent {
	return &agent{
		opts:       newOptions(opts...),
		stop:       make(chan struct{}),
		activities: make([]Activity, 0, maxActivities),
	}
}

// Init initializes the agent with additional options.
func (a *agent) Init(opts ...Option) error {
	for _, o := range opts {
		o(&a.opts)
	}
	return nil
}

// Options returns the current agent options.
func (a *agent) Options() Options {
	return a.opts
}

// String returns the agent name.
func (a *agent) String() string {
	return a.opts.Name
}

// Stop signals the agent to stop its run loop.
func (a *agent) Stop() error {
	a.once.Do(func() {
		close(a.stop)
	})
	return nil
}

// record appends act to the agent's activity log.
// Oldest entries are dropped once the log reaches maxActivities.
func (a *agent) record(act Activity) {
	if act.Time.IsZero() {
		act.Time = time.Now()
	}
	a.actMu.Lock()
	a.activities = append(a.activities, act)
	if len(a.activities) > maxActivities {
		a.activities = a.activities[len(a.activities)-maxActivities:]
	}
	a.actMu.Unlock()
}

// Activity returns a chronological snapshot of recent agent activities.
func (a *agent) Activity() []Activity {
	a.actMu.RLock()
	defer a.actMu.RUnlock()
	result := make([]Activity, len(a.activities))
	copy(result, a.activities)
	return result
}

// Prompt processes a user-provided prompt immediately.
// It is non-blocking: it spawns a goroutine and returns a buffered channel
// that will receive the model response (then be closed). If no model is
// configured, the channel is closed immediately with no value.
func (a *agent) Prompt(text string) <-chan *model.Response {
	ch := make(chan *model.Response, 1)
	a.record(Activity{Type: ActivityPrompt, Prompt: text})
	go func() {
		defer close(ch)
		if a.opts.Model == nil {
			return
		}
		tools := a.buildTools()
		resp, err := a.opts.Model.Generate(a.opts.Context, &model.Request{
			SystemPrompt: a.opts.Directive,
			Prompt:       text,
			Tools:        tools,
		})
		if err != nil {
			a.record(Activity{Type: ActivityError, Prompt: text, Err: err})
			return
		}
		for _, tc := range resp.ToolCalls {
			_, content := a.executeTool(tc.Name, tc.Input)
			if isErrorContent(content) {
				a.record(Activity{Type: ActivityError, Tool: tc.Name, Result: content})
			} else {
				a.record(Activity{Type: ActivityTool, Tool: tc.Name, Result: content})
			}
		}
		reply := resp.Reply
		if reply == "" {
			reply = resp.Answer
		}
		a.record(Activity{Type: ActivityResponse, Prompt: text, Result: reply})
		ch <- resp
	}()
	return ch
}

// Run starts the agent loop. The agent watches the services it manages,
// periodically evaluates their state using the AI model, and acts on
// the results via its built-in service management tools.
func (a *agent) Run() error {
	logger := a.opts.Logger
	logger.Logf(log.InfoLevel, "Starting [agent] %s", a.opts.Name)

	// Build the set of tools available to this agent.
	tools := a.buildTools()

	ticker := time.NewTicker(a.opts.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-a.stop:
			logger.Logf(log.InfoLevel, "Stopping [agent] %s", a.opts.Name)
			return nil
		case <-a.opts.Context.Done():
			return nil
		case <-ticker.C:
			if err := a.evaluate(tools); err != nil {
				logger.Logf(log.ErrorLevel, "[agent] %s evaluate error: %v", a.opts.Name, err)
			}
		}
	}
}

// evaluate asks the model to assess the current state of the managed
// services and execute any necessary management actions.
func (a *agent) evaluate(tools []model.Tool) error {
	if a.opts.Model == nil {
		return nil
	}

	a.record(Activity{Type: ActivityEvaluate})

	status, err := a.serviceStatus()
	if err != nil {
		return err
	}

	prompt := fmt.Sprintf(
		"Current status of managed services: %s\n\nDirective: %s\n\nAssess the services and take any necessary management actions.",
		status, a.opts.Directive,
	)

	req := &model.Request{
		SystemPrompt: a.opts.Directive,
		Prompt:       prompt,
		Tools:        tools,
	}

	resp, err := a.opts.Model.Generate(a.opts.Context, req)
	if err != nil {
		a.record(Activity{Type: ActivityError, Err: err})
		return fmt.Errorf("model generate: %w", err)
	}

	// Execute any tool calls requested by the model.
	for _, tc := range resp.ToolCalls {
		result, content := a.executeTool(tc.Name, tc.Input)
		if isErrorContent(content) {
			a.record(Activity{Type: ActivityError, Tool: tc.Name, Result: content})
		} else {
			a.record(Activity{Type: ActivityTool, Tool: tc.Name, Result: content})
		}
		a.opts.Logger.Logf(log.DebugLevel, "[agent] %s tool %s result: %v", a.opts.Name, tc.Name, result)
	}

	reply := resp.Reply
	if reply == "" {
		reply = resp.Answer
	}
	if reply != "" {
		a.record(Activity{Type: ActivityResponse, Result: reply})
	}

	return nil
}

// serviceStatus returns a JSON summary of the current state of all
// managed services by querying the registry.
func (a *agent) serviceStatus() (string, error) {
	if a.opts.Registry == nil {
		return "{}", nil
	}

	type svcStatus struct {
		Name    string `json:"name"`
		Running bool   `json:"running"`
		Version string `json:"version,omitempty"`
		Nodes   int    `json:"nodes"`
	}

	var statuses []svcStatus

	for _, name := range a.opts.Services {
		svcs, err := a.opts.Registry.GetService(name)
		if err != nil || len(svcs) == 0 {
			statuses = append(statuses, svcStatus{Name: name, Running: false})
			continue
		}
		statuses = append(statuses, svcStatus{
			Name:    name,
			Running: true,
			Version: svcs[0].Version,
			Nodes:   len(svcs[0].Nodes),
		})
	}

	b, err := json.Marshal(statuses)
	if err != nil {
		return "{}", err
	}
	return string(b), nil
}

// buildTools returns the set of model.Tool definitions the agent uses
// to manage its services.
func (a *agent) buildTools() []model.Tool {
	return []model.Tool{
		{
			Name:        "list_services",
			OriginalName: "list_services",
			Description: "List all services managed by this agent along with their current status.",
			Properties: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		{
			Name:        "get_service_status",
			OriginalName: "get_service_status",
			Description: "Get the detailed status of a specific service.",
			Properties: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]any{
						"type":        "string",
						"description": "The name of the service",
					},
				},
				"required": []string{"name"},
			},
		},
		{
			Name:        "call_service",
			OriginalName: "call_service",
			Description: "Make an RPC call to a service endpoint.",
			Properties: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"service": map[string]any{
						"type":        "string",
						"description": "The name of the service to call",
					},
					"endpoint": map[string]any{
						"type":        "string",
						"description": "The endpoint/method to call",
					},
					"request": map[string]any{
						"type":        "object",
						"description": "The request payload",
					},
				},
				"required": []string{"service", "endpoint"},
			},
		},
	}
}

// executeTool dispatches a tool call by name and returns the result.
func (a *agent) executeTool(name string, input map[string]any) (any, string) {
	switch name {
	case "list_services":
		status, err := a.serviceStatus()
		if err != nil {
			return nil, fmt.Sprintf(`{"error": %q}`, err.Error())
		}
		return status, status

	case "get_service_status":
		svcName, _ := input["name"].(string)
		if svcName == "" {
			return nil, `{"error": "name is required"}`
		}
		svcs, err := a.opts.Registry.GetService(svcName)
		if err != nil || len(svcs) == 0 {
			return nil, fmt.Sprintf(`{"name": %q, "running": false}`, svcName)
		}
		b, _ := json.Marshal(map[string]any{
			"name":    svcName,
			"running": true,
			"version": svcs[0].Version,
			"nodes":   len(svcs[0].Nodes),
		})
		return string(b), string(b)

	case "call_service":
		if a.opts.Client == nil {
			return nil, `{"error": "no client configured"}`
		}
		svcName, _ := input["service"].(string)
		endpoint, _ := input["endpoint"].(string)
		if svcName == "" || endpoint == "" {
			return nil, `{"error": "service and endpoint are required"}`
		}

		reqBody, _ := json.Marshal(input["request"])
		req := a.opts.Client.NewRequest(svcName, endpoint, &bytes.Frame{Data: reqBody})
		var rsp bytes.Frame
		if err := a.opts.Client.Call(context.Background(), req, &rsp); err != nil {
			return nil, fmt.Sprintf(`{"error": %q}`, err.Error())
		}
		return string(rsp.Data), string(rsp.Data)

	default:
		// Delegate to a custom tool handler if provided.
		if a.opts.ToolHandler != nil {
			result, content := a.opts.ToolHandler(name, input)
			return result, content
		}
		return nil, fmt.Sprintf(`{"error": "unknown tool %q"}`, name)
	}
}

// isErrorContent reports whether the JSON content string returned by
// executeTool represents a tool error (i.e. contains an "error" key).
func isErrorContent(content string) bool {
	var obj map[string]any
	if err := json.Unmarshal([]byte(content), &obj); err != nil {
		return false
	}
	_, hasErr := obj["error"]
	return hasErr
}

// DefaultAgent is the package-level default Agent instance.
var DefaultAgent Agent

// Run starts the default agent.
func Run() error {
	if DefaultAgent == nil {
		return fmt.Errorf("no default agent configured")
	}
	return DefaultAgent.Run()
}

// NewFunc is a constructor function for creating Agent instances.
type NewFunc func(...Option) Agent

// Directive returns the agent's system prompt / purpose description.
// It is a convenience accessor to Options.Directive.
func Directive(a Agent) string {
	return a.Options().Directive
}

// Services returns the list of service names managed by this agent.
func Services(a Agent) []string {
	return a.Options().Services
}

// WatchServices watches the registry for changes to managed services
// and calls fn whenever a service changes. It blocks until ctx is done.
func WatchServices(ctx context.Context, reg registry.Registry, names []string, fn func(string, *registry.Result)) error {
	if reg == nil {
		return fmt.Errorf("registry is required")
	}

	nameSet := make(map[string]struct{}, len(names))
	for _, n := range names {
		nameSet[n] = struct{}{}
	}

	watcher, err := reg.Watch()
	if err != nil {
		return err
	}

	// Stop the watcher when the context is cancelled so that the
	// blocking Next() call below returns promptly.
	go func() {
		<-ctx.Done()
		watcher.Stop()
	}()

	for {
		res, err := watcher.Next()
		if err != nil {
			// A non-nil error means the watcher was stopped or failed.
			// Return nil when the context was cancelled (expected shutdown).
			select {
			case <-ctx.Done():
				return nil
			default:
				return err
			}
		}
		if res == nil || res.Service == nil {
			continue
		}
		if len(names) == 0 {
			fn(res.Service.Name, res)
			continue
		}
		if _, ok := nameSet[res.Service.Name]; ok {
			fn(res.Service.Name, res)
		}
	}
}
