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
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/urfave/cli/v2"
	"go-micro.dev/v6/agent"
	"go-micro.dev/v6/ai"
	clt "go-micro.dev/v6/client"
	"go-micro.dev/v6/cmd"
	"go-micro.dev/v6/cmd/micro/cli/generate"
	"go-micro.dev/v6/codec/bytes"
	"go-micro.dev/v6/registry"

	_ "go-micro.dev/v6/ai/anthropic"
	_ "go-micro.dev/v6/ai/atlascloud"
	_ "go-micro.dev/v6/ai/gemini"
	_ "go-micro.dev/v6/ai/groq"
	_ "go-micro.dev/v6/ai/mistral"
	_ "go-micro.dev/v6/ai/openai"
	_ "go-micro.dev/v6/ai/together"
)

const systemPromptTmpl = `You are an agent that orchestrates microservices. Use the available tools to fulfill user requests. When you call a tool, explain what you are doing.

Available services: %s

If a user asks for something that no existing service can handle, use the micro_generate_service tool to create it. Pass a short description of what the service should do. After it's created, the new service's endpoints will be available as tools and you can use them immediately.

Do NOT make up capabilities. Only use the tools that are available. If generation fails, tell the user.`

var generateTool = ai.Tool{
	Name:         "micro_generate_service",
	OriginalName: "micro.generate_service",
	Description:  "Generate a new microservice from a description. Use when the user needs a capability that no existing service provides. The service will be created, compiled, and started automatically.",
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
a new service automatically and start using it.

Examples:
  ANTHROPIC_API_KEY=sk-ant-... micro chat --provider anthropic
  micro chat --provider openai --prompt "list all users"`,
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "provider", Usage: "AI provider (anthropic, openai, gemini, groq, mistral, together, atlascloud)", EnvVars: []string{"MICRO_AI_PROVIDER"}},
			&cli.StringFlag{Name: "api_key", Usage: "API key for the provider", EnvVars: []string{"MICRO_AI_API_KEY"}},
			&cli.StringFlag{Name: "model", Usage: "Model name (uses provider default if unset)", EnvVars: []string{"MICRO_AI_MODEL"}},
			&cli.StringFlag{Name: "base_url", Usage: "Override the provider's base URL", EnvVars: []string{"MICRO_AI_BASE_URL"}},
			&cli.StringFlag{Name: "prompt", Usage: "Send a single prompt and exit (non-interactive)"},
		},
		Action: run,
	})
}

// agentInfo holds metadata about a discovered agent.
type agentInfo struct {
	Name     string
	Services []string
}

type session struct {
	provider  string
	apiKey    string
	model     ai.Model
	tools     *ai.Tools
	reg       registry.Registry
	cl        clt.Client
	hist      *ai.History
	toolList  []ai.Tool
	sysPrompt string
	procs     []*exec.Cmd
	agents    map[string]agentInfo

	// Built-in agent capabilities (plan, delegate), shared with the
	// agent package so the direct-service fallback has the same tools a
	// real agent would.
	builtinTools  []ai.Tool
	builtinHandle func(name string, input map[string]any) (any, string, bool)
}

// discoverAgents finds agents registered in the registry.
func (s *session) discoverAgents() bool {
	svcs, err := s.reg.ListServices()
	if err != nil {
		return false
	}

	s.agents = make(map[string]agentInfo)

	for _, svc := range svcs {
		records, err := s.reg.GetService(svc.Name)
		if err != nil || len(records) == 0 {
			continue
		}
		meta := records[0].Metadata
		if meta == nil || meta["type"] != "agent" {
			if len(records[0].Nodes) > 0 {
				meta = records[0].Nodes[0].Metadata
			}
			if meta == nil || meta["type"] != "agent" {
				continue
			}
		}

		var services []string
		if svcsStr := meta["services"]; svcsStr != "" {
			services = strings.Split(svcsStr, ",")
		}

		s.agents[svc.Name] = agentInfo{Name: svc.Name, Services: services}
	}

	return len(s.agents) > 0
}

// callAgent calls an agent's Chat endpoint via RPC.
func (s *session) callAgent(ctx context.Context, name, message string) (*agent.Response, error) {
	reqBody, _ := json.Marshal(map[string]string{"message": message})
	req := s.cl.NewRequest(name, "Agent.Chat", &bytes.Frame{Data: reqBody})
	var rsp bytes.Frame
	if err := s.cl.Call(ctx, req, &rsp); err != nil {
		return nil, err
	}
	var resp struct {
		Reply     string `json:"reply"`
		Agent     string `json:"agent"`
		ToolCalls []struct {
			ID     string `json:"id"`
			Name   string `json:"name"`
			Input  string `json:"input"`
			Result string `json:"result"`
		} `json:"tool_calls"`
	}
	if err := json.Unmarshal(rsp.Data, &resp); err != nil {
		return nil, err
	}
	r := &agent.Response{
		Reply: resp.Reply,
		Agent: resp.Agent,
	}
	for _, tc := range resp.ToolCalls {
		var input map[string]any
		json.Unmarshal([]byte(tc.Input), &input)
		r.ToolCalls = append(r.ToolCalls, ai.ToolCall{
			ID:     tc.ID,
			Name:   tc.Name,
			Input:  input,
			Result: tc.Result,
		})
	}
	return r, nil
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
	discovered, err := s.tools.Discover()
	if err != nil {
		return
	}
	s.toolList = append(discovered, generateTool)
	s.toolList = append(s.toolList, s.builtinTools...)

	serviceNames := make(map[string]bool)
	for _, t := range discovered {
		parts := strings.SplitN(t.OriginalName, ".", 2)
		if len(parts) == 2 {
			serviceNames[parts[0]] = true
		}
	}
	var svcList []string
	for name := range serviceNames {
		svcList = append(svcList, name)
	}
	if len(svcList) == 0 {
		s.sysPrompt = fmt.Sprintf(systemPromptTmpl, "(none yet)")
	} else {
		s.sysPrompt = fmt.Sprintf(systemPromptTmpl, strings.Join(svcList, ", "))
	}
}

func (s *session) handleGenerate(input map[string]any) (any, string) {
	desc, _ := input["description"].(string)
	if desc == "" {
		return map[string]string{"error": "description is required"}, `{"error":"description is required"}`
	}

	fmt.Printf("\n  \033[36m⚡\033[0m generating service: %s\n", desc)

	design, err := generate.Design(context.Background(), s.provider, s.apiKey, "", ".", desc)
	if err != nil {
		msg := fmt.Sprintf(`{"error":"design failed: %s"}`, err)
		return map[string]string{"error": err.Error()}, msg
	}

	if err := generate.Generate(context.Background(), ".", design, s.provider, s.apiKey, ""); err != nil {
		msg := fmt.Sprintf(`{"error":"generate failed: %s"}`, err)
		return map[string]string{"error": err.Error()}, msg
	}

	// Find which services are new (not already in registry)
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
		created = append(created, svc.Name)

		// Build and start the new service
		svcDir, _ := filepath.Abs(svc.Name)
		fmt.Printf("  \033[36m⚡\033[0m starting %s...\n", svc.Name)

		buildCmd := exec.Command("go", "build", "-o", svc.Name, ".")
		buildCmd.Dir = svcDir
		if out, err := buildCmd.CombinedOutput(); err != nil {
			fmt.Printf("  \033[33m⚠\033[0m build failed: %s\n", string(out))
			continue
		}

		runCmd := exec.Command(filepath.Join(svcDir, svc.Name))
		runCmd.Dir = svcDir
		if err := runCmd.Start(); err != nil {
			fmt.Printf("  \033[33m⚠\033[0m start failed: %v\n", err)
			continue
		}
		s.procs = append(s.procs, runCmd)
	}

	if len(created) == 0 {
		result := map[string]any{"message": "No new services needed — all already exist."}
		b, _ := json.Marshal(result)
		return result, string(b)
	}

	// Wait for services to register
	fmt.Printf("  \033[36m⚡\033[0m waiting for services to register...\n")
	time.Sleep(5 * time.Second)

	s.refreshTools()
	fmt.Printf("  \033[32m✓\033[0m %d tools available\n\n", len(s.toolList)-1)

	result := map[string]any{
		"created": created,
		"message": fmt.Sprintf("Created and started: %s. Their endpoints are now available as tools.", strings.Join(created, ", ")),
	}
	b, _ := json.Marshal(result)
	return result, string(b)
}

func run(c *cli.Context) error {
	provider := c.String("provider")
	apiKey := c.String("api_key")
	modelName := c.String("model")
	baseURL := c.String("base_url")
	singlePrompt := c.String("prompt")

	if provider == "" {
		provider = ai.AutoDetectProvider(baseURL)
	}
	if apiKey == "" {
		apiKey = fallbackAPIKey(provider)
	}
	if apiKey == "" {
		return fmt.Errorf("no API key configured; set --api_key or %s", envVarForProvider(provider))
	}

	reg := registry.DefaultRegistry
	cl := clt.DefaultClient

	tools := ai.NewTools(reg, ai.ToolClient(cl))

	// Built-in agent capabilities (plan, delegate), reused from the
	// agent package so the direct-service fallback matches a real agent.
	builtinTools, builtinHandle := agent.Builtins(
		agent.Name("chat"),
		agent.WithRegistry(reg),
		agent.WithClient(cl),
		agent.Provider(provider),
		agent.Model(modelName),
		agent.APIKey(apiKey),
	)

	s := &session{
		provider:      provider,
		apiKey:        apiKey,
		tools:         tools,
		reg:           reg,
		cl:            cl,
		hist:          ai.NewHistory(50),
		builtinTools:  builtinTools,
		builtinHandle: builtinHandle,
	}
	s.refreshTools()

	// Wrap the tool handler to intercept generate calls
	baseHandler := tools.Handler()
	wrappedHandler := func(ctx context.Context, call ai.ToolCall) ai.ToolResult {
		if call.Name == "micro_generate_service" {
			r, c := s.handleGenerate(call.Input)
			return ai.ToolResult{ID: call.ID, Value: r, Content: c}
		}
		if result, content, ok := s.builtinHandle(call.Name, call.Input); ok {
			return ai.ToolResult{ID: call.ID, Value: result, Content: content}
		}
		return baseHandler(ctx, call)
	}

	opts := []ai.Option{
		ai.WithAPIKey(apiKey),
		ai.WithToolHandler(wrappedHandler),
	}
	if modelName != "" {
		opts = append(opts, ai.WithModel(modelName))
	}
	if baseURL != "" {
		opts = append(opts, ai.WithBaseURL(baseURL))
	}

	s.model = ai.New(provider, opts...)
	if s.model == nil {
		return fmt.Errorf("unknown provider: %s", provider)
	}

	defer s.cleanup()

	// Discover registered agents
	hasAgents := s.discoverAgents()

	if singlePrompt != "" {
		return s.ask(c.Context, singlePrompt)
	}

	fmt.Println()
	fmt.Println("  \033[1mmicro chat\033[0m")
	fmt.Println()
	fmt.Printf("  Provider    \033[36m%s\033[0m\n", provider)
	fmt.Printf("  Model       \033[36m%s\033[0m\n", s.model.Options().Model)
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
			return nil
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if line == "exit" || line == "quit" {
			return nil
		}
		if line == "reset" {
			s.hist.Reset()
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
	// If agents are registered, route to them
	if len(s.agents) > 0 {
		return s.routeToAgent(ctx, prompt)
	}

	// Fallback: direct service access (no agents)
	s.hist.Add("user", prompt)

	resp, err := s.model.Generate(ctx, &ai.Request{
		Prompt:       prompt,
		SystemPrompt: s.sysPrompt,
		Tools:        s.toolList,
		Messages:     s.hist.Messages(),
	})
	if err != nil {
		return err
	}

	if resp.Reply != "" {
		s.hist.Add("assistant", resp.Reply)
	}
	if resp.Answer != "" {
		s.hist.Add("assistant", resp.Answer)
	}

	if resp.Reply != "" {
		fmt.Println(resp.Reply)
	}
	for _, tc := range resp.ToolCalls {
		if tc.Name == "micro_generate_service" {
			continue
		}
		args, _ := json.Marshal(tc.Input)
		fmt.Printf("  \033[33m→\033[0m \033[2m%s\033[0m(%s)\n", tc.Name, args)
		if tc.Result != "" {
			fmt.Printf("  \033[32m←\033[0m \033[2m%s\033[0m\n", truncateResult(tc.Result))
		}
		if tc.Error != "" {
			fmt.Printf("  \033[31m✗\033[0m %s\n", tc.Error)
		}
	}
	if resp.Answer != "" {
		fmt.Println()
		fmt.Println(resp.Answer)
	}
	return nil
}

// routeToAgent dispatches a message to the right agent.
// If there's only one agent, sends directly. Otherwise uses the
// LLM to classify intent and route.
func (s *session) routeToAgent(ctx context.Context, prompt string) error {
	// Single agent — call directly via RPC
	if len(s.agents) == 1 {
		for name := range s.agents {
			fmt.Printf("  \033[35m◆\033[0m \033[2m%s\033[0m\n", name)
			resp, err := s.callAgent(ctx, name, prompt)
			if err != nil {
				return err
			}
			s.printAgentResponse(resp)
			return nil
		}
	}

	// Multiple agents — use LLM to route
	routeTool := ai.Tool{
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

	routerHandler := func(ctx context.Context, call ai.ToolCall) ai.ToolResult {
		agentName, _ := call.Input["agent"].(string)
		message, _ := call.Input["message"].(string)
		if message == "" {
			message = prompt
		}

		if _, ok := s.agents[agentName]; !ok {
			return ai.ToolResult{ID: call.ID, Value: map[string]string{"error": "unknown agent: " + agentName}, Content: `{"error":"unknown agent"}`}
		}

		fmt.Printf("  \033[35m◆\033[0m \033[2m%s\033[0m\n", agentName)
		resp, err := s.callAgent(ctx, agentName, message)
		if err != nil {
			return ai.ToolResult{ID: call.ID, Value: map[string]string{"error": err.Error()}, Content: `{"error":"` + err.Error() + `"}`}
		}

		s.printAgentResponse(resp)

		result := map[string]any{"agent": agentName, "reply": resp.Reply}
		b, _ := json.Marshal(result)
		return ai.ToolResult{ID: call.ID, Value: result, Content: string(b)}
	}

	routerModel := ai.New(s.provider,
		ai.WithAPIKey(s.apiKey),
		ai.WithToolHandler(routerHandler),
	)

	resp, err := routerModel.Generate(ctx, &ai.Request{
		Prompt:       prompt,
		SystemPrompt: s.buildRouterPrompt(),
		Tools:        []ai.Tool{routeTool},
	})
	if err != nil {
		return err
	}

	if resp.Answer != "" {
		fmt.Println()
		fmt.Println(resp.Answer)
	}

	return nil
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

func (s *session) cleanup() {
	for _, p := range s.procs {
		if p.Process != nil {
			p.Process.Kill()
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
