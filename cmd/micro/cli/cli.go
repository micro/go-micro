package microcli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/urfave/cli/v2"
	"go-micro.dev/v6/client"
	"go-micro.dev/v6/cmd"
	"go-micro.dev/v6/codec/bytes"
	"go-micro.dev/v6/registry"

	"go-micro.dev/v6/cmd/micro/cli/new"
	"go-micro.dev/v6/cmd/micro/cli/util"

	// Import packages that register commands via init()
	_ "go-micro.dev/v6/cmd/micro/cli/agent"
	_ "go-micro.dev/v6/cmd/micro/cli/build"
	_ "go-micro.dev/v6/cmd/micro/cli/deploy"
	_ "go-micro.dev/v6/cmd/micro/cli/init"
	_ "go-micro.dev/v6/cmd/micro/cli/remote"
)

const zeroToHeroHelp = `0→hero no-secret lifecycle demo

Run this from a go-micro repository checkout when you want one command that
proves the maintained services → agents → workflows path without provider keys:

  ./internal/harness/zero-to-hero-ci/run.sh

That script runs the same deterministic path CI uses:
  - CLI discovery for scaffold, run, chat, inspect, flow runs, and deploy dry-run
  - the smallest first-agent example
  - the support-desk reference app with services, an agent, a flow, and an approval gate
  - plan/delegate and universe harnesses with only the model mocked

If you only want the runnable examples first:
  go run ./examples/first-agent
  go run ./examples/support

Full local contract:
  make harness

Guide: https://go-micro.dev/docs/guides/zero-to-hero.html`

const examplesWayfinding = `First-agent examples (no provider key required)

Run these from a go-micro repository checkout in this order:

  1. Smallest service-backed agent
       go run ./examples/first-agent
     Proves an agent can call a service tool with the deterministic mock model.

  2. No-secret support-agent transcript
       go test ./internal/harness/zero-to-hero-ci -run TestNoSecretFirstAgentTranscript -count=1
     Exercises service tools, mock-model chat, and inspectable run history.

  3. Full services → agents → workflows reference app
       go run ./examples/support
     Shows the support desk service, agent, workflow, and approval gate together.

Then continue the same path with the installed CLI:
  micro agent demo
  micro docs
  micro zero-to-hero

Guides:
  https://go-micro.dev/docs/guides/no-secret-first-agent.html
  https://go-micro.dev/docs/guides/your-first-agent.html
  https://go-micro.dev/docs/guides/debugging-agents.html
  https://go-micro.dev/docs/guides/zero-to-hero.html`

const docsWayfinding = `First-agent and 0→hero docs:

  1. Start with the no-secret CLI demo
       micro agent demo
     This prints the maintained support-agent transcript command so you can
     prove service tools, mock-model chat, and inspectable run history without
     configuring a provider key.

  2. No-secret first-agent transcript
     https://go-micro.dev/docs/guides/no-secret-first-agent.html
     Run the maintained support agent without a provider key:
       go test ./internal/harness/zero-to-hero-ci -run TestNoSecretFirstAgentTranscript -count=1

  3. Your First Agent
     https://go-micro.dev/docs/guides/your-first-agent.html
     Build a service-backed agent, then use:
       micro agent preflight  # before micro run: prerequisites
       micro run
       micro chat
       micro agent doctor     # after micro run: chat/gateway/inspect recovery

  4. Debugging your agent
     https://go-micro.dev/docs/guides/debugging-agents.html
     Inspect agent runs and memory with:
       micro agent doctor
       micro inspect agent <name>
       micro agent history <name>

  5. 0→hero Reference
     https://go-micro.dev/docs/guides/zero-to-hero.html
     Walk the scaffold → run → chat → inspect → deploy dry-run lifecycle.`

func genProtoHandler(c *cli.Context) error {
	cmd := exec.Command("find", ".", "-name", "*.proto", "-exec", "protoc", "--proto_path=.", "--micro_out=.", "--go_out=.", `{}`, `;`)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func init() {
	cmd.Register([]*cli.Command{
		{
			Name:      "new",
			Usage:     "Create a new service",
			ArgsUsage: "[name]",
			UsageText: `  micro new helloworld                          # scaffold a single service
  micro new --prompt "a todo list with tasks"    # AI-design multiple services
  micro new --prompt "add tags to the task service"  # extend existing services`,
			Action: new.Run,
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:  "no-mcp",
					Usage: "Disable MCP gateway integration in generated code",
				},
				&cli.BoolFlag{
					Name:  "proto",
					Usage: "Use Protocol Buffers (requires protoc); default is reflection-based, no protoc needed",
				},
				&cli.StringFlag{
					Name:  "template",
					Usage: "Service template: default, crud, pubsub, api",
				},
				&cli.StringFlag{
					Name:    "prompt",
					Usage:   "Describe the system to generate (uses AI to design & build services with real business logic)",
					EnvVars: []string{"MICRO_NEW_PROMPT"},
				},
				&cli.StringFlag{
					Name:    "provider",
					Usage:   "AI provider for --prompt (anthropic, openai, gemini, atlascloud, groq, mistral, together)",
					EnvVars: []string{"MICRO_AI_PROVIDER"},
				},
				&cli.StringFlag{
					Name:    "api_key",
					Usage:   "API key for --prompt (or set ANTHROPIC_API_KEY, OPENAI_API_KEY, etc.)",
					EnvVars: []string{"MICRO_AI_API_KEY"},
				},
			},
		},
		{
			Name:  "gen",
			Usage: "Generate various things",
			Subcommands: []*cli.Command{
				{
					Name:   "proto",
					Usage:  "Generate proto requires protoc and protoc-gen-micro",
					Action: genProtoHandler,
				},
			},
		},
		{
			Name:  "services",
			Usage: "List available services",
			Action: func(ctx *cli.Context) error {
				services, err := registry.ListServices()
				if err != nil {
					return err
				}
				for _, service := range services {
					fmt.Println(service.Name)
				}
				return nil
			},
		},

		{
			Name:  "examples",
			Usage: "Show provider-free first-agent example paths",
			Description: `Print the maintained no-secret examples for the services → agents →
workflows on-ramp: first-agent, transcript, support app, and matching guides.`,
			Action: func(ctx *cli.Context) error {
				fmt.Fprintln(ctx.App.Writer, examplesWayfinding)
				return nil
			},
		},
		{
			Name:  "zero-to-hero",
			Usage: "Show the no-secret 0→hero lifecycle demo command",
			Description: `Print the maintained provider-free services → agents → workflows
lifecycle command and the smaller runnable examples it covers.`,
			Aliases: []string{"hero"},
			Action: func(ctx *cli.Context) error {
				fmt.Fprintln(ctx.App.Writer, zeroToHeroHelp)
				return nil
			},
		},
		{
			Name:  "docs",
			Usage: "Show the first-agent and 0→hero documentation path",
			Description: `Print the maintained adoption on-ramp for new Go Micro developers:
the no-secret first-agent transcript, Your First Agent, debugging guide, and
0→hero lifecycle reference.`,
			Action: func(ctx *cli.Context) error {
				fmt.Fprintln(ctx.App.Writer, docsWayfinding)
				return nil
			},
		},
		{
			Name:  "call",
			Usage: "Call a service",
			Flags: []cli.Flag{
				&cli.StringSliceFlag{
					Name:    "header",
					Aliases: []string{"H"},
					Usage:   "Set request headers (can be used multiple times): --header 'Key:Value'",
				},
				&cli.StringSliceFlag{
					Name:    "metadata",
					Aliases: []string{"m"},
					Usage:   "Set request metadata (can be used multiple times): --metadata 'Key:Value'",
				},
			},
			Action: func(ctx *cli.Context) error {
				args := ctx.Args()

				if args.Len() < 2 {
					return fmt.Errorf("usage: [service] [endpoint] [request]")
				}

				service := args.Get(0)
				endpoint := args.Get(1)
				request := `{}`

				if args.Len() == 3 {
					request = args.Get(2)
				}

				// Create context with metadata if provided
				// Note: This is for the direct 'micro call' command.
				// Dynamic service calls (e.g., 'micro helloworld call') are handled in CallService.
				callCtx := context.TODO()
				callCtx = util.AddMetadataToContext(callCtx, ctx.StringSlice("metadata"))
				callCtx = util.AddMetadataToContext(callCtx, ctx.StringSlice("header"))

				req := client.NewRequest(service, endpoint, &bytes.Frame{Data: []byte(request)})
				var rsp bytes.Frame
				err := client.Call(callCtx, req, &rsp)
				if err != nil {
					return err
				}

				fmt.Print(string(rsp.Data))
				return nil
			},
		},
		{
			Name:  "describe",
			Usage: "Describe a service",
			Action: func(ctx *cli.Context) error {
				args := ctx.Args()

				if args.Len() != 1 {
					return fmt.Errorf("usage: [service]")
				}

				service := args.Get(0)
				services, err := registry.GetService(service)
				if err != nil {
					return err
				}
				if len(services) == 0 {
					return nil
				}
				b, _ := json.MarshalIndent(services[0], "", "    ")
				fmt.Println(string(b))
				return nil
			},
		},
		// Note: The following commands are registered in their respective packages:
		// - status, logs, stop: remote/remote.go
		// - build: build/build.go
		// - deploy: deploy/deploy.go
		// - init: init/init.go
	}...)

	cmd.App().Action = func(c *cli.Context) error {
		if c.Args().Len() == 0 {
			return nil
		}

		v, err := exec.LookPath("micro-" + c.Args().First())
		if err == nil {
			ce := exec.Command(v, c.Args().Slice()[1:]...)
			ce.Stdout = os.Stdout
			ce.Stderr = os.Stderr
			return ce.Run()
		}

		command := c.Args().Get(0)
		args := c.Args().Slice()

		if srv, err := util.LookupService(command); err != nil {
			return util.CliError(err)
		} else if srv != nil && util.ShouldRenderHelp(args) {
			return cli.Exit(util.FormatServiceUsage(srv, c), 0)
		} else if srv != nil {
			err := util.CallService(srv, args)
			return util.CliError(err)
		}

		return nil
	}
}
