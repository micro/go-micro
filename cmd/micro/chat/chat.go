// Package chat implements the 'micro chat' interactive agent command.
//
// micro chat opens a terminal REPL where you can talk to your services
// through an LLM. It discovers all services from the registry, exposes
// each endpoint as a tool, and lets the model orchestrate calls in
// response to natural-language prompts.
package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/urfave/cli/v2"
	"go-micro.dev/v6/agent"
	agentpb "go-micro.dev/v6/agent/proto"
	agentweb "go-micro.dev/v6/agent/web"
	"go-micro.dev/v6/agent/workspace"
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
List available skills with workspace_skill before starting work. Load relevant skills
on demand. Read project files before changing them; reads include the AGENTS.md
instructions for their directory. Nested instructions apply to files beneath them. Use workspace tools to inspect, edit and
verify work. Respect project instructions. Use micro_generate_service when a new
service is needed. Use only available tools and report failures honestly.
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

With one registered agent, micro chat connects directly without a local API key.
Use micro chat NAME to select a particular agent when several are running.
With no registered agents, chat uses a local development agent.

micro chat discovers every service in the registry, exposes each endpoint as a
tool alongside workspace and web tools. Describe the work you need done.
The model decides which tool to call and
issues RPCs to the right service.

The local agent can read, search, edit files and run commands in the current
project. It reads AGENTS.md and asks before tool actions that can change state.
It can also generate a missing service; its tools are available on your next message.

Examples:
  ANTHROPIC_API_KEY=sk-ant-... micro chat --provider anthropic
  micro chat --provider openai --prompt "list all users"
  micro chat --prompt "create a task"`,
		ArgsUsage: "[agent]",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "sandbox", Usage: "Run workspace commands in this Docker image with networking disabled"},
			&cli.StringFlag{Name: "host", Usage: "Serve the local development agent under this name without a chat UI (loopback only); --yes allows unattended tool actions"},
			&cli.StringFlag{Name: "provider", Usage: "AI provider (anthropic, openai, gemini, groq, mistral, together, atlascloud)", EnvVars: []string{"MICRO_AI_PROVIDER"}},
			&cli.StringFlag{Name: "api_key", Usage: "API key for the provider", EnvVars: []string{"MICRO_AI_API_KEY"}},
			&cli.StringFlag{Name: "model", Usage: "Model name (uses provider default if unset)", EnvVars: []string{"MICRO_AI_MODEL"}},
			&cli.StringFlag{Name: "base_url", Usage: "Override the provider's base URL", EnvVars: []string{"MICRO_AI_BASE_URL"}},
			&cli.StringFlag{Name: "session", Usage: "Resume a conversation by ID"},
			&cli.BoolFlag{Name: "yes", Usage: "Allow local tool actions without prompting (including shell commands)"},
			&cli.BoolFlag{Name: "new", Usage: "Start a new conversation"},
			&cli.StringFlag{Name: "prompt", Usage: "Send a single prompt and exit (non-interactive)"},
			&cli.BoolFlag{Name: "stream", Usage: "Show agent tool events and answer chunks"},
		},
		Action: run,
	})
}

// agentInfo holds metadata about a discovered agent.
type agentInfo struct {
	Interaction bool
	Schedules   bool
	Tasks       bool
	History     bool
	Provider    string
	Model       string
	Name        string
	Services    []string
	Stream      bool
	Sessions    bool
}

type session struct {
	display       *chatDisplay
	workspace     *workspace.Workspace
	instructions  string
	approvals     chan approvalRequest
	approvalMu    sync.Mutex
	yes           bool
	interactiveUI bool
	hosting       bool
	id            string
	state         store.Store
	output        io.Writer
	project       string

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

// newHarness uses the selected conversation for memory and plan state.
// Tests without a backing store keep their state in memory.
func (s *session) newHarness(name, prompt string, opts ...agent.Option) agent.Agent {
	base := []agent.Option{
		agent.Name(name), agent.Prompt(prompt), agent.Provider(s.provider),
		agent.Model(s.modelName), agent.APIKey(s.apiKey), agent.BaseURL(s.baseURL),
		agent.WithRegistry(s.reg), agent.WithClient(s.cl),
		agent.WithStore(store.NewMemoryStore()),
		agent.CompactMemory(50, 20),
		agent.ModelCallTimeout(5 * time.Minute), agent.ToolCallTimeout(2 * time.Minute),
	}
	if s.state != nil {
		base = append(base, agent.WithStore(s.state), agent.Session(s.localSessionID()))
	}
	return agent.New(append(base, opts...)...)
}

func (s *session) developmentAgent() agent.Agent {
	if s.local == nil {
		generate := s.generate
		if generate == nil {
			generate = s.handleGenerate
		}
		opts := []agent.Option{agent.WithTool(generateTool.Name, generateTool.Description, generateTool.Properties, generate)}
		if s.state != nil && !s.hosting {
			opts = append(opts, s.memorySearchTool())
		}
		if s.workspace != nil {
			opts = append(opts, s.workspace.Tools()...)
			web := agentweb.Web{}
			if key := os.Getenv("BRAVE_SEARCH_API_KEY"); key != "" {
				web.Search = agentweb.Brave(key)
			}
			opts = append(opts, web.Tools()...)
			opts = append(opts, agent.WithApproval(s.approve))
		}
		s.local = s.newHarness("micro-chat", systemPrompt+s.instructions, opts...)
	}
	return s.local
}

func (s *session) reset() { s.local, s.router = nil, nil }

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

		info := agentInfo{Interaction: true, History: true, Tasks: true, Schedules: true, Provider: meta["provider"], Model: meta["model"], Name: svc.Name, Services: services, Stream: true, Sessions: true}
		for _, record := range records {
			// Registries may combine old and new nodes during a rolling deploy.
			// Service metadata cannot establish the capability of each target node.
			if len(record.Nodes) == 0 {
				info.Interaction = false
				info.Sessions = false
				info.History = false
				info.Tasks = false
				info.Schedules = false
			}
			for _, node := range record.Nodes {
				if node == nil || node.Metadata["interaction"] != "v1" {
					info.Interaction = false
				}
				if node == nil || node.Metadata["schedules"] != "v1" {
					info.Schedules = false
				}
				if node == nil || node.Metadata["tasks"] != "v1" {
					info.Tasks = false
				}
				if node == nil || node.Metadata["session_history"] != "v1" {
					info.History = false
				}
				if node == nil || node.Metadata["sessions"] != "v1" {
					info.Sessions = false
				}
			}
			supportsStream := false
			for _, endpoint := range record.Endpoints {
				if endpoint != nil && endpoint.Name == "Agent.StreamChat" {
					supportsStream = true
				}
			}
			info.Stream = info.Stream && supportsStream
		}
		s.agents[svc.Name] = info
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
		_ = json.Unmarshal([]byte(tc.Input), &input)
		r.ToolCalls = append(r.ToolCalls, model.ToolCall{
			ID:     tc.ID,
			Name:   tc.Name,
			Input:  input,
			Result: tc.Result,
		})
	}
	return r, nil
}

// streamAgent calls an agent's StreamChat endpoint and prints chunks as they
// arrive. Callers check endpoint metadata before choosing this path; an error
// after dispatch must not replay potentially completed work through Agent.Chat.
func (s *session) streamAgent(ctx context.Context, name, message string) error {
	if s.agents[name].Interaction {
		return s.interactAgent(ctx, name, message)
	}
	stream, err := agentpb.NewAgentService(name, s.cl).StreamChat(ctx, &agentpb.ChatRequest{Message: message})
	if err != nil {
		return err
	}
	defer stream.Close()
	if s.display != nil {
		defer s.display.flush()
	}
	for {
		chunk, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		if chunk == nil || chunk.Reply == "" {
			continue
		}

		if s.display != nil {
			s.display.token(chunk.Reply)
		} else {
			fmt.Fprint(s.writer(), chunk.Reply)
		}
	}
	if s.display == nil {
		fmt.Fprintln(s.writer())
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
		s.toolList = discovered
		if len(s.agents) == 0 {
			s.toolList = append(s.toolList, generateTool)
		}
	}
}

func (s *session) handleGenerate(ctx context.Context, input map[string]any) (string, error) {
	desc, _ := input["description"].(string)
	if strings.TrimSpace(desc) == "" {
		return "", fmt.Errorf("description is required")
	}
	fmt.Fprintf(s.writer(), "\n  Generating service: %s\n", desc)
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

	reg := *cmd.DefaultOptions().Registry
	cl := *cmd.DefaultOptions().Client

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
	if c.String("host") != "" {
		s.agents = nil
		hasAgents = false
		targetAgent = ""
	}
	if targetAgent != "" {
		if _, ok := s.agents[targetAgent]; !ok {
			return fmt.Errorf("agent %q is not registered; run `micro agent list` to see available agents", targetAgent)
		}
		s.agents = map[string]agentInfo{targetAgent: s.agents[targetAgent]}
		hasAgents = true
	}
	if err := s.openSession(c.String("session"), c.Bool("new")); err != nil {
		return err
	}
	defer func() { _ = s.state.Close() }()
	// Remote agents own their model configuration.
	if len(s.agents) != 1 {
		if err := s.configure(c); err != nil {
			return err
		}
	}
	if !hasAgents {
		var err error
		skillsHome, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		s.workspace, err = workspace.New(".", workspace.WithContainer(c.String("sandbox")), workspace.WithSkills(filepath.Join(skillsHome, ".agents", "skills")), workspace.WithOutput(agent.ToolOutput))
		if err != nil {
			return err
		}
		instructions, err := s.workspace.Instructions()
		if err != nil {
			return fmt.Errorf("project instructions: %w", err)
		}
		if instructions != "" {
			s.instructions = "\n\nProject instructions (AGENTS.md):\n" + instructions
		}
	}
	s.yes = c.Bool("yes")
	s.approvals = make(chan approvalRequest)
	s.refreshTools()

	defer s.cleanup()

	if name := c.String("host"); name != "" {
		s.hosting = true
		// Release the CLI settings database before serving so another chat process
		// can open it. The host only needs agent state from this point onward.
		if err := s.state.Close(); err != nil {
			return err
		}
		s.state = store.NewFileStore()
		host := s.developmentAgent()
		host.Init(agent.Name(name), agent.Session(""), agent.Address("127.0.0.1:0"))
		ctx, stop := signal.NotifyContext(c.Context, os.Interrupt, syscall.SIGTERM)
		defer stop()
		done := make(chan error, 1)
		go func() { done <- host.Run() }()
		select {
		case err := <-done:
			return err
		case <-ctx.Done():
			_ = host.Stop()
			return <-done
		}
	}

	if singlePrompt != "" {
		return s.ask(c.Context, singlePrompt)
	}

	fmt.Fprintln(s.writer())
	fmt.Fprintln(s.writer(), "  \033[1mmicro chat\033[0m")
	fmt.Fprintln(s.writer())
	if len(s.agents) != 1 {
		fmt.Fprintf(s.writer(), "  Provider    \033[36m%s\033[0m\n", s.provider)
		fmt.Fprintf(s.writer(), "  Model       \033[36m%s\033[0m\n", s.modelName)
		fmt.Fprintln(s.writer())
	}
	if hasAgents {
		fmt.Fprintln(s.writer(), "  Agents:")
		for name, info := range s.agents {
			fmt.Fprintf(s.writer(), "    \033[35m◆\033[0m %s \033[2m(%s)\033[0m\n", name, strings.Join(info.Services, ", "))
		}
		fmt.Fprintln(s.writer())
	}
	if !hasAgents {
		fmt.Fprintln(s.writer(), "  Tools:")
		for _, name := range []string{"workspace_skill", "workspace_read", "workspace_search", "workspace_write", "workspace_edit", "workspace_exec"} {
			fmt.Fprintf(s.writer(), "    ● %s\n", name)
		}
		for _, t := range s.toolList {
			fmt.Fprintf(s.writer(), "    \033[32m●\033[0m %s\n", t.OriginalName)
		}
	}
	if len(s.toolList) == 0 && !hasAgents {
		fmt.Fprintln(s.writer(), "    \033[33m(no services found)\033[0m")
	}
	fmt.Fprintln(s.writer())
	fmt.Fprintln(s.writer(), "  Type a prompt and press enter. \033[2mCtrl-D or 'exit' to quit.\033[0m")
	fmt.Fprintln(s.writer())

	return s.interactive(c.Context)
}

func (s *session) ask(ctx context.Context, prompt string) error {
	if s.id != "" {
		for name, info := range s.agents {
			if !info.Sessions {
				return fmt.Errorf("agent %q does not support isolated conversations; rebuild it with the current Go Micro version", name)
			}
		}
		ctx = agent.WithSession(ctx, s.id)
	}
	if s.state != nil {
		if err := s.remember(prompt); err != nil {
			return err
		}
	}
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
	if s.display != nil {
		defer s.display.flush()
	}
	for {
		event, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			if s.display == nil {
				fmt.Fprintln(s.writer())
			}
			return nil
		}
		if err != nil {
			return err
		}
		if event == nil {
			continue
		}
		s.showEvent(event)
	}
}

// routeToAgent dispatches a message to the right agent.
// If there's only one agent, sends directly. Otherwise uses the
// LLM to classify intent and route.
func (s *session) routeToAgent(ctx context.Context, prompt string) error {
	// Single agent — call directly via RPC
	if len(s.agents) == 1 {
		for name := range s.agents {
			fmt.Fprintf(s.writer(), "  \033[35m◆\033[0m \033[2m%s\033[0m\n", name)
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
			fmt.Fprintf(s.writer(), "  ◆ %s\n", name)
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
		fmt.Fprintf(s.writer(), "    \033[33m→\033[0m \033[2m%s\033[0m(%s)\n", tc.Name, args)
		if tc.Result != "" {
			fmt.Fprintf(s.writer(), "    \033[32m←\033[0m \033[2m%s\033[0m\n", truncateResult(tc.Result))
		}
	}
	if resp.Reply != "" {
		fmt.Fprintln(s.writer())
		fmt.Fprintln(s.writer(), resp.Reply)
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
	if s.workspace != nil {
		_ = s.workspace.Close()
	}
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

func (s *session) showEvent(event *agent.StreamEvent) {
	switch event.Type {
	case agent.StreamEventToolStart:
		if s.display != nil {
			s.display.flush()
		}
		if event.ToolCall.Name == "delegate" {
			fmt.Fprintf(s.writer(), "  → delegate to %v: %v\n", terminalText(fmt.Sprint(event.ToolCall.Input["to"])), terminalText(fmt.Sprint(event.ToolCall.Input["task"])))
		} else {
			fmt.Fprintf(s.writer(), "  → %s\n", terminalText(event.ToolCall.Name))
		}
	case agent.StreamEventToolEnd:
		if s.display != nil {
			s.display.flush()
		}
		fmt.Fprintf(s.writer(), "  ← %s\n", terminalText(truncateResult(event.Result.Content)))
	case agent.StreamEventToolOutput:
		if s.display != nil {
			s.display.output(event.Token)
		} else {
			fmt.Fprint(s.writer(), event.Token)
		}
	case agent.StreamEventToken:
		if s.display != nil {
			s.display.token(event.Token)
		} else {
			fmt.Fprint(s.writer(), event.Token)
		}
	}
}
