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
	"strings"

	"github.com/urfave/cli/v2"
	"go-micro.dev/v5/ai"
	"go-micro.dev/v5/client"
	"go-micro.dev/v5/cmd"
	"go-micro.dev/v5/registry"

	// Side-effect imports register the AI providers.
	_ "go-micro.dev/v5/ai/anthropic"
	_ "go-micro.dev/v5/ai/atlascloud"
	_ "go-micro.dev/v5/ai/gemini"
	_ "go-micro.dev/v5/ai/groq"
	_ "go-micro.dev/v5/ai/mistral"
	_ "go-micro.dev/v5/ai/openai"
	_ "go-micro.dev/v5/ai/together"
)

const systemPromptTmpl = `You are an agent that orchestrates microservices. Use the available tools to fulfill user requests. When you call a tool, explain what you are doing.

Available services: %s

If a user asks for something that no existing service can handle, tell them which service they need and suggest the command to create it. For example: "You don't have a shipping service yet. Run: micro new --prompt 'add a shipping service' to create one."

Do NOT make up capabilities. Only use the tools that are available.`

func init() {
	cmd.Register(&cli.Command{
		Name:  "chat",
		Usage: "Interactive AI chat that orchestrates your services",
		Description: `Start an interactive chat session that uses an LLM to call your services.

micro chat discovers every service in the registry, exposes each endpoint as a
tool, and lets you ask natural-language questions like "list all users" or
"create an order for product 42". The model decides which tool to call and
issues RPCs to the right service.

Examples:
  # Chat with Anthropic Claude (uses ANTHROPIC_API_KEY)
  ANTHROPIC_API_KEY=sk-ant-... micro chat --provider anthropic

  # Use a single prompt and exit
  micro chat --provider openai --prompt "list all users"

  # Use a custom provider via base URL (auto-detected)
  micro chat --api_key $KEY --base_url https://api.groq.com/openai

Environment variables:
  MICRO_AI_PROVIDER      Provider name (anthropic, openai, gemini, groq, ...)
  MICRO_AI_API_KEY       API key for the provider
  MICRO_AI_MODEL         Model name override
  MICRO_AI_BASE_URL      Base URL override
  ANTHROPIC_API_KEY      Fallback API key for the anthropic provider
  OPENAI_API_KEY         Fallback API key for the openai provider
  GEMINI_API_KEY         Fallback API key for the gemini provider`,
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

func run(c *cli.Context) error {
	provider := c.String("provider")
	apiKey := c.String("api_key")
	model := c.String("model")
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

	// Discover tools and wire up a handler that routes to the right RPC.
	reg := registry.DefaultRegistry
	cli := client.DefaultClient

	tools := ai.NewTools(reg, ai.ToolClient(cli))
	discovered, err := tools.Discover()
	if err != nil {
		return fmt.Errorf("discover tools: %w", err)
	}

	opts := []ai.Option{
		ai.WithAPIKey(apiKey),
		ai.WithTools(tools),
	}
	if model != "" {
		opts = append(opts, ai.WithModel(model))
	}
	if baseURL != "" {
		opts = append(opts, ai.WithBaseURL(baseURL))
	}

	m := ai.New(provider, opts...)
	if m == nil {
		return fmt.Errorf("unknown provider: %s", provider)
	}

	hist := ai.NewHistory(50)

	// Build service list for system prompt
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
	sysPrompt := fmt.Sprintf(systemPromptTmpl, strings.Join(svcList, ", "))

	if singlePrompt != "" {
		return ask(c.Context, m, hist, discovered, sysPrompt, singlePrompt)
	}

	// Startup banner
	fmt.Println()
	fmt.Println("  \033[1mmicro chat\033[0m")
	fmt.Println()
	fmt.Printf("  Provider    \033[36m%s\033[0m\n", provider)
	fmt.Printf("  Model       \033[36m%s\033[0m\n", m.Options().Model)
	fmt.Println()
	fmt.Println("  Tools:")
	for _, t := range discovered {
		fmt.Printf("    \033[32m●\033[0m %s\n", t.OriginalName)
	}
	if len(discovered) == 0 {
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
			hist.Reset()
			fmt.Println("\033[2m(history cleared)\033[0m")
			fmt.Println()
			continue
		}
		if err := ask(c.Context, m, hist, discovered, sysPrompt, line); err != nil {
			fmt.Printf("\033[31merror:\033[0m %v\n", err)
		}
		fmt.Println()
	}
}

func ask(ctx context.Context, m ai.Model, hist *ai.History, toolList []ai.Tool, sysPrompt, prompt string) error {
	hist.Add("user", prompt)

	resp, err := m.Generate(ctx, &ai.Request{
		Prompt:       prompt,
		SystemPrompt: sysPrompt,
		Tools:        toolList,
		Messages:     hist.Messages(),
	})
	if err != nil {
		return err
	}

	if resp.Reply != "" {
		hist.Add("assistant", resp.Reply)
	}
	if resp.Answer != "" {
		hist.Add("assistant", resp.Answer)
	}

	if resp.Reply != "" {
		fmt.Println(resp.Reply)
	}
	for _, tc := range resp.ToolCalls {
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

// fallbackAPIKey returns the provider-specific environment variable when
// neither --api_key nor MICRO_AI_API_KEY is set. This lets users keep
// existing ANTHROPIC_API_KEY / OPENAI_API_KEY / GEMINI_API_KEY vars
// without re-exporting them.
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
