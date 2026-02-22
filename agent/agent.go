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
}

// agent is the default Agent implementation.
type agent struct {
	opts Options
	stop chan struct{}
	once sync.Once
}

// New creates a new Agent with the given options.
func New(opts ...Option) Agent {
	return &agent{
		opts: newOptions(opts...),
		stop: make(chan struct{}),
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
		return fmt.Errorf("model generate: %w", err)
	}

	// Execute any tool calls requested by the model.
	for _, tc := range resp.ToolCalls {
		result, _ := a.executeTool(tc.Name, tc.Input)
		a.opts.Logger.Logf(log.DebugLevel, "[agent] %s tool %s result: %v", a.opts.Name, tc.Name, result)
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
