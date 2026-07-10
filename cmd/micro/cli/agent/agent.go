// Package agent registers the 'micro agent' CLI commands.
package agent

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/urfave/cli/v2"
	goagent "go-micro.dev/v6/agent"
	"go-micro.dev/v6/cmd"
	"go-micro.dev/v6/registry"
	"go-micro.dev/v6/store"
)

const firstAgentQuickChecksHelp = `First-agent failure-mode quick checks

Use this when scaffold -> run -> chat -> inspect stalls and you want the
smallest provider-free recovery loop before reading the full docs.

1. Confirm prerequisites before starting the gateway:
   micro agent preflight

2. Start the project and keep it running in a separate terminal:
   micro run

3. Check the agent is registered and the chat gateway is reachable:
   micro agent doctor

4. If chat returns an answer or an error, inspect the latest run state:
   micro inspect agent <name>
   micro runs <name>

5. If provider chat is not configured yet, prove the no-secret path still works:
   micro agent demo
   go test ./internal/harness/zero-to-hero-ci -run TestNoSecretFirstAgentTranscript -count=1

Recovery docs:
  https://go-micro.dev/docs/guides/debugging-agents.html
  https://go-micro.dev/docs/guides/no-secret-first-agent.html`

const noSecretDemoHelp = `No-secret first-agent demo

Use this when you want the fastest provider-free agent success path before
configuring API keys. It runs the maintained support/first-agent transcript with
the deterministic mock model used by CI:

  go test ./internal/harness/zero-to-hero-ci -run TestNoSecretFirstAgentTranscript -count=1

What this proves:
  - service tools can be called by an agent
  - chat behavior is exercised without contacting a live provider
  - run history can be inspected after the prompt

After it passes:
  - Build your own service-backed agent: https://go-micro.dev/docs/guides/your-first-agent.html
  - Diagnose provider-backed chat:     https://go-micro.dev/docs/guides/debugging-agents.html
  - Walk the full 0→hero lifecycle:    https://go-micro.dev/docs/guides/zero-to-hero.html

Use live-provider chat when you are ready for real model behavior:
  micro agent preflight  # before micro run: prerequisites
  micro run
  micro chat
  micro agent doctor     # after micro run: chat/gateway/inspect recovery
  micro inspect agent <name>`

func init() {
	cmd.Register(&cli.Command{
		Name:      "runs",
		Usage:     "Show recorded agent runs",
		ArgsUsage: "[agent] [run-id]",
		Flags:     runFlags(),
		Action: func(c *cli.Context) error {
			name := c.Args().First()
			if name == "" {
				return fmt.Errorf("usage: micro runs [agent] [run-id]")
			}
			if runID := c.Args().Get(1); runID != "" {
				return printRunHistory(name, runID, c.Bool("json"))
			}
			return printRunIndex(name, runOptions(c), c.Bool("json"))
		},
	})

	cmd.Register(&cli.Command{
		Name:  "agent",
		Usage: "Manage AI agents (try: micro agent demo)",
		Subcommands: []*cli.Command{
			{
				Name:  "demo",
				Usage: "Show the no-secret first-agent demo command",
				Description: `Print the provider-free first-agent path for new developers:
the deterministic mock-model transcript, when to use it, and where to go next
for live-provider chat and inspect/debugging.`,
				Action: func(c *cli.Context) error {
					fmt.Fprintln(c.App.Writer, noSecretDemoHelp)
					return nil
				},
			},
			{
				Name:    "quickcheck",
				Aliases: []string{"debug"},
				Usage:   "Print first-agent failure-mode quick checks",
				Description: `Print provider-free recovery breadcrumbs for the scaffold -> run ->
chat -> inspect loop, including exact commands for registration, gateway, run
history, and no-secret fallback checks.`,
				Action: func(c *cli.Context) error {
					fmt.Fprintln(c.App.Writer, firstAgentQuickChecksHelp)
					return nil
				},
			},
			{
				Name:  "preflight",
				Usage: "Check local prerequisites before the first provider-backed agent",
				Action: func(c *cli.Context) error {
					return runAgentPreflight(os.Stdout, defaultPreflightDeps())
				},
			},
			{
				Name:  "doctor",
				Usage: "Diagnose chat, gateway, registration, provider, and inspect recovery after micro run",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "gateway", Value: "http://localhost:8080", Usage: "Gateway URL started by micro run"},
				},
				Action: func(c *cli.Context) error {
					return runAgentDoctor(os.Stdout, defaultDoctorDeps(), c.String("gateway"))
				},
			},
			{
				Name:  "list",
				Usage: "List registered agents",
				Action: func(c *cli.Context) error {
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
						if meta == nil || meta["type"] != "agent" {
							if len(records[0].Nodes) > 0 {
								meta = records[0].Nodes[0].Metadata
							}
							if meta == nil || meta["type"] != "agent" {
								continue
							}
						}
						found = true
						services := meta["services"]
						if services == "" {
							services = "(all)"
						}
						fmt.Printf("  \033[35m◆\033[0m %-20s manages: %s\n", svc.Name, services)
					}
					if !found {
						fmt.Println("  No agents registered.")
						fmt.Println()
						fmt.Println("  Start an agent with:")
						fmt.Println("    micro run  (if agents are part of your project)")
					}
					return nil
				},
			},
			{
				Name:      "describe",
				Usage:     "Describe an agent",
				ArgsUsage: "[name]",
				Action: func(c *cli.Context) error {
					name := c.Args().First()
					if name == "" {
						return fmt.Errorf("usage: micro agent describe [name]")
					}
					records, err := registry.GetService(name)
					if err != nil {
						return err
					}
					if len(records) == 0 {
						return fmt.Errorf("agent %s not found", name)
					}
					b, _ := json.MarshalIndent(records[0], "", "  ")
					fmt.Println(string(b))
					return nil
				},
			},
			{
				Name:      "history",
				Usage:     "Show an agent's stored conversation and run history",
				ArgsUsage: "[name] [run-id]",
				Flags:     runFlags(),
				Action: func(c *cli.Context) error {
					name := c.Args().First()
					if name == "" {
						return fmt.Errorf("usage: micro agent history [name] [run-id]")
					}
					if runID := c.Args().Get(1); runID != "" {
						return printRunHistory(name, runID, c.Bool("json"))
					}
					if c.Bool("json") {
						return printRunIndex(name, runOptions(c), true)
					}
					// Read from the agent's scoped state store (database
					// "agent", table = name) — available whether or not the
					// agent is currently running.
					mem := goagent.NewMemory(store.Scope(store.DefaultStore, "agent", name), "history", 1000)
					msgs := mem.Messages()
					if len(msgs) == 0 {
						fmt.Printf("  No history for agent %q.\n", name)
					} else {
						for _, m := range msgs {
							fmt.Printf("  \033[2m%s:\033[0m %v\n", m.Role, m.Content)
						}
					}
					return printRunIndex(name, runOptions(c), c.Bool("json"))
				},
			},
		},
	})
}

func runFlags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{Name: "json", Usage: "Print run data as JSON for automation"},
		&cli.StringFlag{Name: "status", Usage: "Only show runs with this status (running, done, error, refused)"},
		&cli.StringFlag{Name: "trace", Usage: "Only show runs whose trace id matches this full id or prefix"},
		&cli.IntFlag{Name: "limit", Usage: "Show the most recently updated N runs"},
	}
}

func runOptions(c *cli.Context) goagent.RunListOptions {
	return goagent.RunListOptions{Status: c.String("status"), TraceID: c.String("trace"), Limit: c.Int("limit")}
}

func printRunIndex(name string, opts goagent.RunListOptions, asJSON bool) error {
	runs, err := goagent.ListRunSummariesWithOptions(store.DefaultStore, name, opts)
	if err != nil {
		return err
	}
	return writeRunIndex(os.Stdout, name, runs, asJSON)
}

func writeRunIndex(w io.Writer, name string, runs []goagent.RunSummary, asJSON bool) error {
	if asJSON {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(runs)
	}
	if len(runs) == 0 {
		fmt.Fprintf(w, "  No runs recorded for agent %q.\n", name)
		return nil
	}
	fmt.Fprintln(w, "  Runs:")
	for _, run := range runs {
		line := fmt.Sprintf("    %s  status=%s  events=%d  duration=%s  last=%s  updated=%s", run.RunID, run.Status, run.Events, formatDurationMS(run.DurationMS), run.LastKind, run.UpdatedAt.Format("2006-01-02 15:04:05"))
		if run.ParentID != "" {
			line += "  parent=" + run.ParentID
		}
		if run.TraceID != "" {
			line += "  trace=" + shortTraceID(run.TraceID)
		}
		if run.Checkpoint != "" {
			line += "  checkpoint=" + run.Checkpoint
		}
		if run.Stage != "" {
			line += "  stage=" + run.Stage
		}
		if run.LastError != "" {
			line += "  error=" + run.LastError
		}
		fmt.Fprintln(w, line)
		writeRunIndexBreadcrumbs(w, name, run)
	}
	return nil
}

func writeRunIndexBreadcrumbs(w io.Writer, name string, run goagent.RunSummary) {
	if run.Stage == "input-required" {
		fmt.Fprintf(w, "      inspect: micro agent history %s %s\n", name, run.RunID)
		fmt.Fprintf(w, "      input:   call micro.AgentResumeInput(ctx, agent, %q, input) to continue the input-required run\n", run.RunID)
		return
	}
	if !isResumableRunSummary(run) {
		return
	}
	fmt.Fprintf(w, "      inspect: micro agent history %s %s\n", name, run.RunID)
	fmt.Fprintf(w, "      resume:  call micro.AgentResume(ctx, agent, %q) after recreating the agent with the same checkpoint store\n", run.RunID)
	fmt.Fprintf(w, "      stream:  call micro.ResumeStreamAsk(ctx, agent, %q) to resume with streaming events\n", run.RunID)
}

func isResumableRunSummary(run goagent.RunSummary) bool {
	switch run.Status {
	case "running", "error", "failed", "refused":
		return run.Checkpoint != "done" || run.Stage != ""
	default:
		return false
	}
}

func printRunHistory(name, runID string, asJSON bool) error {
	events, err := goagent.LoadRunEvents(store.DefaultStore, name, runID)
	if err != nil {
		return err
	}
	return writeRunHistory(os.Stdout, name, runID, events, asJSON)
}

func writeRunHistory(w io.Writer, name, runID string, events []goagent.RunEvent, asJSON bool) error {
	if asJSON {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(events)
	}
	if len(events) == 0 {
		fmt.Fprintf(w, "  No run %q for agent %q.\n", runID, name)
		return nil
	}
	for _, e := range events {
		line := fmt.Sprintf("  %s %-5s", e.Time.Format("15:04:05.000"), e.Kind)
		if e.Name != "" {
			line += " " + e.Name
		}
		if e.Provider != "" || e.Model != "" {
			line += fmt.Sprintf(" %s/%s", e.Provider, e.Model)
		}
		if e.LatencyMS > 0 {
			line += fmt.Sprintf(" %dms", e.LatencyMS)
		}
		if e.Tokens.TotalTokens > 0 {
			line += fmt.Sprintf(" tokens=%d", e.Tokens.TotalTokens)
		}
		if e.ParentID != "" {
			line += " parent=" + e.ParentID
		}
		if e.TraceID != "" {
			line += " trace=" + shortTraceID(e.TraceID)
		}
		if e.Refused != "" {
			line += " refused=" + e.Refused
		}
		if e.Error != "" {
			line += " error=" + e.Error
		}
		fmt.Fprintln(w, line)
	}
	return nil
}

func formatDurationMS(ms int64) string {
	if ms <= 0 {
		return "0ms"
	}
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	return fmt.Sprintf("%.1fs", float64(ms)/1000)
}

func shortTraceID(id string) string {
	if len(id) <= 12 {
		return id
	}
	return id[:12]
}
