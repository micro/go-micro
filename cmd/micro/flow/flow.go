// Package flow implements the 'micro flow' command for event-driven
// LLM orchestration of microservices.
package flow

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/urfave/cli/v2"
	aiflow "go-micro.dev/v5/flow"
	"go-micro.dev/v5/broker"
	"go-micro.dev/v5/client"
	"go-micro.dev/v5/cmd"
	"go-micro.dev/v5/registry"
)

func init() {
	cmd.Register(&cli.Command{
		Name:  "flow",
		Usage: "Event-driven LLM orchestration",
		Description: `Run flows that subscribe to broker events and use an LLM to
orchestrate service calls in response.

Examples:
  # Run a flow that reacts to user creation events
  micro flow run --trigger events.user.created \
    --prompt "New user: {{.Data}}. Send welcome email." \
    --provider anthropic

  # Run a one-shot flow with inline data
  micro flow exec --prompt "List all users and count them" \
    --provider anthropic

  # Run a flow with a specific model
  micro flow exec --prompt "Create a test user" \
    --provider atlascloud --model deepseek-ai/DeepSeek-V3-0324`,
		Subcommands: []*cli.Command{
			{
				Name:  "run",
				Usage: "Start a flow that listens to broker events",
				Flags: flowFlags(),
				Action: func(c *cli.Context) error {
					return runFlow(c, false)
				},
			},
			{
				Name:  "exec",
				Usage: "Execute a flow once with inline data",
				Flags: append(flowFlags(), &cli.StringFlag{
					Name:  "data",
					Usage: "Input data for the flow (default: reads from --prompt only)",
				}),
				Action: func(c *cli.Context) error {
					return runFlow(c, true)
				},
			},
		},
	})
}

func flowFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{Name: "trigger", Usage: "Broker topic to subscribe to", EnvVars: []string{"MICRO_FLOW_TRIGGER"}},
		&cli.StringFlag{Name: "prompt", Usage: "Prompt template (use {{.Data}} for event data)", EnvVars: []string{"MICRO_FLOW_PROMPT"}},
		&cli.StringFlag{Name: "provider", Usage: "AI provider", Value: "openai", EnvVars: []string{"MICRO_AI_PROVIDER"}},
		&cli.StringFlag{Name: "api_key", Usage: "API key", EnvVars: []string{"MICRO_AI_API_KEY"}},
		&cli.StringFlag{Name: "model", Usage: "Model name", EnvVars: []string{"MICRO_AI_MODEL"}},
		&cli.StringFlag{Name: "base_url", Usage: "Provider base URL", EnvVars: []string{"MICRO_AI_BASE_URL"}},
		&cli.StringFlag{Name: "name", Usage: "Flow name", Value: "default"},
	}
}

func runFlow(c *cli.Context, oneShot bool) error {
	prompt := c.String("prompt")
	if prompt == "" {
		return fmt.Errorf("--prompt is required")
	}

	provider := c.String("provider")
	apiKey := c.String("api_key")
	if apiKey == "" {
		apiKey = fallbackKey(provider)
	}
	if apiKey == "" {
		return fmt.Errorf("no API key; set --api_key or the provider's env var")
	}

	opts := []aiflow.Option{
		aiflow.Prompt(prompt),
		aiflow.Provider(provider),
		aiflow.APIKey(apiKey),
	}
	if v := c.String("trigger"); v != "" {
		opts = append(opts, aiflow.Trigger(v))
	}
	if v := c.String("model"); v != "" {
		opts = append(opts, aiflow.Model(v))
	}
	if v := c.String("base_url"); v != "" {
		opts = append(opts, aiflow.BaseURL(v))
	}

	opts = append(opts, aiflow.OnResult(func(r aiflow.Result) {
		out, _ := json.MarshalIndent(r, "", "  ")
		fmt.Println(string(out))
	}))

	f := aiflow.New(c.String("name"), opts...)

	reg := registry.DefaultRegistry
	br := broker.DefaultBroker
	cl := client.DefaultClient

	if err := br.Connect(); err != nil {
		return fmt.Errorf("broker connect: %w", err)
	}

	if err := f.Register(reg, br, cl); err != nil {
		return err
	}

	if oneShot {
		data := c.String("data")
		if data == "" {
			data = prompt
		}
		return f.Execute(context.Background(), data)
	}

	if c.String("trigger") == "" {
		return fmt.Errorf("--trigger is required for 'flow run' (use 'flow exec' for one-shot)")
	}

	fmt.Println()
	fmt.Println("  \033[1mmicro flow\033[0m")
	fmt.Println()
	fmt.Printf("  Flow       \033[36m%s\033[0m\n", f.Name())
	fmt.Printf("  Topic      \033[36m%s\033[0m\n", c.String("trigger"))
	fmt.Printf("  Provider   \033[36m%s\033[0m\n", provider)
	fmt.Println()
	fmt.Println("  \033[2mListening for events. Ctrl-C to stop.\033[0m")
	fmt.Println()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	fmt.Printf("\nStopped. %d executions recorded.\n", len(f.Results()))
	return nil
}

func fallbackKey(provider string) string {
	envMap := map[string]string{
		"anthropic":  "ANTHROPIC_API_KEY",
		"openai":     "OPENAI_API_KEY",
		"gemini":     "GEMINI_API_KEY",
		"groq":       "GROQ_API_KEY",
		"mistral":    "MISTRAL_API_KEY",
		"together":   "TOGETHER_API_KEY",
		"atlascloud": "ATLASCLOUD_API_KEY",
	}
	if env, ok := envMap[provider]; ok {
		return os.Getenv(env)
	}
	return ""
}
