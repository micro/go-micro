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
	"go-micro.dev/v6/broker"
	"go-micro.dev/v6/client"
	"go-micro.dev/v6/cmd"
	aiflow "go-micro.dev/v6/flow"
	"go-micro.dev/v6/registry"
)

func init() {
	cmd.Register(&cli.Command{
		Name:  "flow",
		Usage: "Event-driven LLM orchestration",
		Description: `Run flows that subscribe to broker events and use an LLM to
orchestrate service calls in response.

This command runs a single-step flow from flags. For ordered, durable
multi-step workflows (checkpointed steps that resume after a crash),
define the flow in code with micro.FlowSteps and a Checkpoint — see
examples/flow-durable.

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
			{
				Name:   "list",
				Usage:  "List running flows (from the registry, type=flow)",
				Action: listFlows,
			},
			{
				Name:      "runs",
				Usage:     "Show durable run history for a flow",
				ArgsUsage: "[name]",
				Action:    flowRuns,
			},
		},
	})
}

// listFlows shows flows currently registered in the registry — the live
// view, mirroring `micro agent list`.
func listFlows(c *cli.Context) error {
	svcs, err := registry.ListServices()
	if err != nil {
		return err
	}
	found := false
	for _, svc := range svcs {
		records, err := registry.GetService(svc.Name)
		if err != nil || len(records) == 0 {
			continue
		}
		meta := records[0].Metadata
		isFlow := meta != nil && meta["type"] == "flow"
		if !isFlow && len(records[0].Nodes) > 0 {
			nm := records[0].Nodes[0].Metadata
			isFlow = nm != nil && nm["type"] == "flow"
		}
		if !isFlow {
			continue
		}
		found = true
		trigger := ""
		if meta != nil {
			trigger = meta["trigger"]
		}
		fmt.Printf("  \033[36m⚡\033[0m %-20s trigger: %s\n", svc.Name, trigger)
	}
	if !found {
		fmt.Println("  No running flows.")
		fmt.Println("  Run one with: micro flow run --trigger <topic> --prompt <...>")
	}
	return nil
}

// flowRuns shows a flow's durable run history from the store — the
// historic view, available whether or not the flow is currently running.
func flowRuns(c *cli.Context) error {
	name := c.Args().First()
	if name == "" {
		return fmt.Errorf("flow name required: micro flow runs <name>")
	}
	runs, err := aiflow.StoreCheckpoint(nil, name).List(context.Background())
	if err != nil {
		return err
	}
	if len(runs) == 0 {
		fmt.Printf("  No runs recorded for flow %q.\n", name)
		return nil
	}
	for _, r := range runs {
		id := r.ID
		if len(id) > 8 {
			id = id[:8]
		}
		stage := r.State.Stage
		if stage == "" {
			stage = "-"
		}
		fmt.Printf("  %s  %-8s stage=%-12s (%d steps)\n", id, r.Status, stage, len(r.Steps))
	}
	return nil
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
