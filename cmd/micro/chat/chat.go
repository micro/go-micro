// Package chat implements the 'micro chat' interactive agent command.
//
// micro chat opens a terminal REPL where you can talk to your services
// through an LLM. It discovers all services from the registry, exposes
// each endpoint as a tool, and lets the model orchestrate calls in
// response to natural-language prompts.
package chat

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/urfave/cli/v2"
	"go-micro.dev/v6/agent"
	agentpb "go-micro.dev/v6/agent/proto"
	clt "go-micro.dev/v6/client"
	"go-micro.dev/v6/cmd"
	"go-micro.dev/v6/cmd/micro/cli/generate"
	"go-micro.dev/v6/codec/bytes"
	"go-micro.dev/v6/model"
	"go-micro.dev/v6/registry"
	"go-micro.dev/v6/store"

	_ "go-micro.dev/v6/model/anthropic"
	_ "go-micro.dev/v6/model/atlascloud"
	_ "go-micro.dev/v6/model/gemini"
	_ "go-micro.dev/v6/model/groq"
	_ "go-micro.dev/v6/model/mistral"
	_ "go-micro.dev/v6/model/openai"
	_ "go-micro.dev/v6/model/together"
)

const systemPrompt = `You are a development agent that uses microservices to fulfill requests.
Use only the available tools. If a capability is missing, use micro_generate_service.
New services become available as tools on the next user message. Report generation
failures honestly and ask the user to continue after creating a service.`

var generateTool = model.Tool{
	Name:         "micro_generate_service",
	OriginalName: "micro.generate_service",
	Description:  "Generate a new microservice from a description. Use when the user needs a capability that no existing service provides. The service will be created, compiled, and started automatically; its tools are available on the next user message.",
	Properties: map[string]any{
		"description": map[string]any{
			"type":        "string",
			"description": "What the service should do, e.g. 'a shipping service that tracks parcels and calculates rates'",
		},
	},
}

func init() {
	cmd.Register(&cli.Command{
		Name:  "chat",
		Usage: "Interactive AI chat that orchestrates your services",
		Description: `Start an interactive chat session that uses an LLM to call your services.

micro chat discovers every service in the registry, exposes each endpoint as a
tool, and lets you ask natural-language questions like "list all users" or
"create an order for product 42". The model decides which tool to call and
issues RPCs to the right service.

If you ask for something no existing service handles, the agent will generate
a new service automatically. Its tools are available on your next message.

Examples:
  ANTHROPIC_API_KEY=sk-ant-... micro chat --provider anthropic
  micro chat --provider openai --prompt "list all users"
  micro chat assistant --prompt "create a task"`,
		ArgsUsage: "[agent]",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "provider", Usage: "AI provider (anthropic, openai, gemini, groq, mistral, together, atlascloud)", EnvVars: []string{"MICRO_AI_PROVIDER"}},
			&cli.StringFlag{Name: "api_key", Usage: "API key for the provider", EnvVars: []string{"MICRO_AI_API_KEY"}},
			&cli.StringFlag{Name: "model", Usage: "Model name (uses provider default if unset)", EnvVars: []string{"MICRO_AI_MODEL"}},
			&cli.StringFlag{Name: "base_url", Usage: "Override the provider's base URL", EnvVars: []string{"MICRO_AI_BASE_URL"}},
			&cli.StringFlag{Name: "prompt", Usage: "Send a single prompt and exit (non-interactive)"},
			&cli.BoolFlag{Name: "stream", Usage: "Show agent tool events and answer chunks"},
		},
		Action: run,
	})
}

// agentInfo holds metadata about a discovered agent.
type agentInfo struct {
	Name     string
	Services []string
	Stream   bool
}

type session struct {
	provider  string
	apiKey    string
	modelName string
	baseURL   string
	reg       registry.Registry
	cl        clt.Client
	toolList  []model.Tool
	procs     []*exec.Cmd
	procMu    sync.Mutex
	closed    bool
	agents    map[string]agentInfo
	stream    bool
	local     agent.Agent
	router    agent.Agent
	generate  agent.ToolFunc
}

// newHarness supplies session-local state; it never registers a CLI agent or
// persists conversations/plans under the identity of another chat session.
func (s *session) newHarness(name, prompt string, opts ...agent.Option) agent.Agent {
	base := []agent.Option{
		agent.Name(name), agent.Prompt(prompt), agent.Provider(s.provider),
		agent.Model(s.modelName), agent.APIKey(s.apiKey), agent.BaseURL(s.baseURL),
		agent.WithRegistry(s.reg), agent.WithClient(s.cl),
		agent.WithStore(store.NewMemoryStore()), agent.WithMemory(agent.NewInMemory(50)),
		agent.ModelCallTimeout(5 * time.Minute), agent.ToolCallTimeout(2 * time.Minute),
	}
	return agent.New(append(base, opts...)...)
}

func (s *session) developmentAgent() agent.Agent {
	if s.local == nil {
		generate := s.generate
		if generate == nil {
			generate = s.handleGenerate
		}
		s.local = s.newHarness("micro-chat", systemPrompt,
			agent.WithTool(generateTool.Name, generateTool.Description, generateTool.Properties, generate))
	}
	return s.local
}

func (s *session) reset() { s.local, s.router = nil, nil }

// discoverAgents finds agents registered in the registry.
func (s *session) discoverAgents() bool {
	svcs, err := s.reg.ListServices()
	if err != nil {
		return fa…695 tokens truncated… := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		if chunk == nil || chunk.Reply == "" {
			continue
		}
		fmt.Print(chunk.Reply)
		reply.WriteString(chunk.Reply)
	}
	if reply.Len() > 0 {
		fmt.Println()
	}
	return nil
}

// buildRouterPrompt creates a system prompt for the router that
// knows about all available agents and can dispatch to them.
func (s *session) buildRouterPrompt() string {
	var agentDescs []string
	for name, info := range s.agents {
		svcs := strings.Join(info.Services, ", ")
		agentDescs = append(agentDescs, fmt.Sprintf("- %s (manages: %s)", name, svcs))
	}
	sort.Strings(agentDescs)

	return fmt.Sprintf(`You are a router that dispatches user requests to the right agent.

Available agents:
%s

For each user message, decide which agent should handle it and call the route_to_agent tool with the agent name and the message. If the request spans multiple agents, call route_to_agent multiple times.

If no agent can handle the request, say so.`, strings.Join(agentDescs, "\n"))
}

func (s *session) refreshTools() {
	discovered, err := model.NewTools(s.reg, model.ToolClient(s.cl)).Discover()
	if err == nil {
		s.toolList = append(discovered, generateTool)
	}
}

func (s *session) handleGenerate(ctx context.Context, input map[string]any) (string, error) {
	desc, _ := input["description"].(string)
	if strings.TrimSpace(desc) == "" {
		return "", fmt.Errorf("description is required")
	}
	fmt.Printf("\n  Generating service: %s\n", desc)
	design, err := generate.Design(ctx, s.provider, s.apiKey, s.modelName, ".", desc)
	if err != nil {
		return "", fmt.Errorf("design failed: %w", err)
	}
	if err := generate.Generate(ctx, ".", design, s.provider, s.apiKey, s.modelName); err != nil {
		return "", fmt.Errorf("generate failed: %w", err)
	}
	existing := make(map[string]bool)
	if svcs, err := s.reg.ListServices(); err == nil {
		for _, svc := range svcs {
			existing[svc.Name] = true
		}
	}
	var created []string
	for _, svc := range design.Services {
		name := strings.TrimSuffix(svc.Name, "-service")
		if existing[name] {
			continue
		}
		if err := ctx.Err(); err != nil {
			return "", err
		}
		svcDir, err := filepath.Abs(svc.Name)
		if err != nil {
			return "", err
		}
		build := exec.CommandContext(ctx, "go", "build", "-o", svc.Name, ".")
		build.Dir = svcDir
		if out, err := build.CombinedOutput(); err != nil {
			return "", fmt.Errorf("build %s: %w: %s", svc.Name, err, out)
		}
		process := exec.Command(filepath.Join(svcDir, svc.Name))
		process.Dir = svcDir
		if err := s.startProcess(ctx, process); err != nil {
			return "", fmt.Errorf("start %s: %w", svc.Name, err)
		}
		created = append(created, name)
	}
	if len(created) == 0 {
		return `{"message":"No new services needed; the services already exist."}`, nil
	}
	wait, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	for _, name := range created {
		for {
			records, err := s.reg.GetService(name)
			ready := false
			if err == nil {
				for _, record := range records {
					ready = ready || len(record.Nodes) > 0
				}
			}
			if ready {
				break
			}
			select {
			case <-wait.Done():
				return "", fmt.Errorf("waiting for %s registration: %w", name, wait.Err())
			case <-time.After(100 * time.Millisecond):
			}
		}
	}
	result, err := json.Marshal(map[string]any{"created": created, "message": "Services are running. Their tools will be discovered on the next user message."})
	return string(result), err
}

func run(c *cli.Context) error {
	provider := c.String("provider")
	apiKey := c.String("api_key")
	modelName := c.String("model")
	baseURL := c.String("base_url")
	singlePrompt := c.String("prompt")
	streamOutput := c.Bool("stream")
	targetAgent := c.Args().First()

	if provider == "" {
		provider = model.AutoDetectProvider(baseURL)
	}
	if apiKey == "" {
		apiKey = fallbackAPIKey(provider)
	}

	reg := registry.DefaultRegistry
	cl := clt.DefaultClient

	s := &session{
		provider:  provider,
		apiKey:    apiKey,
		modelName: modelName,
		baseURL:   baseURL,
		reg:       reg,
		cl:        cl,
		stream:    streamOutput,
	}
	hasAgents := s.discoverAgents()
	if targetAgent != "" {
		if _, ok := s.agents[targetAgent]; !ok {
			return fmt.Errorf("agent %q is not registered; run `micro agent list` to see available agents", targetAgent)
		}
		s.agents = map[string]agentInfo{targetAgent: s.agents[targetAgent]}
		hasAgents = true
	}
	if targetAgent != "" && singlePrompt != "" {
		return s.ask(c.Context, singlePrompt)
	}
	if apiKey == "" {
		return fmt.Errorf("no API key configured; set --api_key or %s", envVarForProvider(provider))
	}

	// Constructing the adapter resolves defaults without running a model call.
	configured := model.New(provider, model.WithAPIKey(apiKey), model.WithModel(modelName), model.WithBaseURL(baseURL))
	if configured == nil {
		return fmt.Errorf("unknown provider: %s", provider)
	}
	s.modelName = configured.Options().Model
	s.refreshTools()

	defer s.cleanup()

	if singlePrompt != "" {
		return s.ask(c.Context, singlePrompt)
	}

	fmt.Println()
	fmt.Println("  \033[1mmicro chat\033[0m")
	fmt.Println()
	fmt.Printf("  Provider    \033[36m%s\033[0m\n", provider)
	fmt.Printf("  Model       \033[36m%s\033[0m\n", s.modelName)
	fmt.Println()
	if hasAgents {
		fmt.Println("  Agents:")
		for name, info := range s.agents {
			fmt.Printf("    \033[35m◆\033[0m %s \033[2m(%s)\033[0m\n", name, strings.Join(info.Services, ", "))
		}
		fmt.Println()
	}
	fmt.Println("  Tools:")
	for _, t := range s.toolList {
		fmt.Printf("    \033[32m●\033[0m %s\n", t.OriginalName)
	}
	if len(s.toolList) == 0 && !hasAgents {
		fmt.Println("    \033[33m(no services found)\033[0m")
	}
	fmt.Println()
	fmt.Println("  Type a prompt and press enter. \033[2mCtrl-D or 'exit' to quit.\033[0m")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, 4096), 1024*1024)
	for {
		fmt.Print("\033[1;36m>\033[0m ")
		if !scanner.Scan() {
			fmt.Println()
			return scanner.Err()
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if line == "exit" || line == "quit" {
			return nil
		}
		if line == "reset" {
			s.reset()
			fmt.Println("\033[2m(history cleared)\033[0m")
			fmt.Println()
			continue
		}
		if err := s.ask(c.Context, line); err != nil {
			fmt.Printf("\033[31merror:\033[0m %v\n", err)
		}
		fmt.Println()
	}
}

func (s *session) ask(ctx context.Context, prompt string) error {
	if len(s.agents) > 0 {
		return s.routeToAgent(ctx, prompt)
	}
	return s.askHarness(ctx, s.developmentAgent(), prompt)
}

func (s *session) askHarness(ctx context.Context, ag agent.Agent, prompt string) error {
	if !s.stream {
		response, err := ag.Ask(ctx, prompt)
		if err != nil {
			return err
		}
		s.printAgentResponse(response)
		return nil
	}
	stream, err := agent.StreamAsk(ctx, ag, prompt)
	if err != nil {
		return err
	}
	defer stream.Close()
	for {
		event, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			fmt.Println()
			return nil
		}
		if err != nil {
			return err
		}
		if event == nil {
			continue
		}
		switch event.Type {
		case agent.StreamEventToolStart:
			fmt.Printf("  → %s\n", event.ToolCall.Name)
		case agent.StreamEventToolEnd:
			fmt.Printf("  ← %s\n", truncateResult(event.Result.Content))
		case agent.StreamEventToken:
			fmt.Print(event.Token)
		}
	}
}

// routeToAgent dispatches a message to the right agent.
// If there's only one agent, sends directly. Otherwise uses the
// LLM to classify intent and route.
func (s *session) routeToAgent(ctx context.Context, prompt string) error {
	// Single agent — call directly via RPC
	if len(s.agents) == 1 {
		for name := range s.agents {
			fmt.Printf("  \033[35m◆\033[0m \033[2m%s\033[0m\n", name)
			if s.stream && s.agents[name].Stream {
				return s.streamAgent(ctx, name, prompt)
			}
			resp, err := s.callAgent(ctx, name, prompt)
			if err != nil {
				return err
			}
			s.printAgentResponse(resp)
			return nil
		}
	}

	// Multiple agents — use LLM to route
	routeTool := model.Tool{
		Name:         "route_to_agent",
		OriginalName: "route_to_agent",
		Description:  "Route a message to a specific agent for handling.",
		Properties: map[string]any{
			"agent": map[string]any{
				"type":        "string",
				"description": "The agent name to route to",
			},
			"message": map[string]any{
				"type":        "string",
				"description": "The message to send to the agent",
			},
		},
	}

	if s.router == nil {
		route := func(ctx context.Context, input map[string]any) (string, error) {
			name, _ := input["agent"].(string)
			message, _ := input["message"].(string)
			if _, ok := s.agents[name]; !ok {
				return "", fmt.Errorf("unknown agent: %s", name)
			}
			if strings.TrimSpace(message) == "" {
				return "", fmt.Errorf("message is required")
			}
			fmt.Printf("  ◆ %s\n", name)
			response, err := s.callAgent(ctx, name, message)
			if err != nil {
				return "", err
			}
			s.printAgentResponse(response)
			result, err := json.Marshal(map[string]string{"agent": name, "reply": response.Reply})
			return string(result), err
		}
		s.router = s.newHarness("micro-chat-router", s.buildRouterPrompt(), agent.Services(),
			agent.WithTool(routeTool.Name, routeTool.Description, routeTool.Properties, route))
	}
	return s.askHarness(ctx, s.router, prompt)
}

func (s *session) printAgentResponse(resp *agent.Response) {
	for _, tc := range resp.ToolCalls {
		args, _ := json.Marshal(tc.Input)
		fmt.Printf("    \033[33m→\033[0m \033[2m%s\033[0m(%s)\n", tc.Name, args)
		if tc.Result != "" {
			fmt.Printf("    \033[32m←\033[0m \033[2m%s\033[0m\n", truncateResult(tc.Result))
		}
	}
	if resp.Reply != "" {
		fmt.Println()
		fmt.Println(resp.Reply)
	}
}

func (s *session) startProcess(ctx context.Context, process *exec.Cmd) error {
	s.procMu.Lock()
	defer s.procMu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	if s.closed {
		return errors.New("chat session closed")
	}
	if err := process.Start(); err != nil {
		return err
	}
	s.procs = append(s.procs, process)
	return nil
}

func (s *session) cleanup() {
	s.procMu.Lock()
	s.closed = true
	processes := s.procs
	s.procs = nil
	s.procMu.Unlock()
	for _, p := range processes {
		if p.Process != nil {
			_ = p.Process.Kill()
			_ = p.Wait()
		}
	}
}

func fallbackAPIKey(provider string) string {
	if v := os.Getenv(envVarForProvider(provider)); v != "" {
		return v
	}
	return ""
}

func envVarForProvider(provider string) string {
	switch provider {
	case "anthropic":
		return "ANTHROPIC_API_KEY"
	case "openai":
		return "OPENAI_API_KEY"
	case "gemini":
		return "GEMINI_API_KEY"
	case "groq":
		return "GROQ_API_KEY"
	case "mistral":
		return "MISTRAL_API_KEY"
	case "together":
		return "TOGETHER_API_KEY"
	case "atlascloud":
		return "ATLASCLOUD_API_KEY"
	default:
		return "MICRO_AI_API_KEY"
	}
}

func truncateResult(s string) string {
	if len(s) <= 200 {
		return s
	}
	return s[:200] + "..."
}
