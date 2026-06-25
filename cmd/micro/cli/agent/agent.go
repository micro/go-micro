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

func init() {
	cmd.Register(&cli.Command{
		Name:      "runs",
		Usage:     "Show recorded agent runs",
		ArgsUsage: "[agent] [run-id]",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "json", Usage: "Print run data as JSON for automation"},
		},
		Action: func(c *cli.Context) error {
			name := c.Args().First()
			if name == "" {
				return fmt.Errorf("usage: micro runs [agent] [run-id]")
			}
			if runID := c.Args().Get(1); runID != "" {
				return printRunHistory(name, runID, c.Bool("json"))
			}
			return printRunIndex(name, c.Bool("json"))
		},
	})

	cmd.Register(&cli.Command{
		Name:  "agent",
		Usage: "Manage AI agents",
		Subcommands: []*cli.Command{
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
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "json", Usage: "Print run data as JSON for automation"},
				},
				Action: func(c *cli.Context) error {
					name := c.Args().First()
					if name == "" {
						return fmt.Errorf("usage: micro agent history [name] [run-id]")
					}
					if runID := c.Args().Get(1); runID != "" {
						return printRunHistory(name, runID, c.Bool("json"))
					}
					if c.Bool("json") {
						return printRunIndex(name, true)
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
					return printRunIndex(name, c.Bool("json"))
				},
			},
		},
	})
}

func printRunIndex(name string, asJSON bool) error {
	runs, err := goagent.ListRunSummaries(store.DefaultStore, name)
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
		line := fmt.Sprintf("    %s  events=%d  last=%s  updated=%s", run.RunID, run.Events, run.LastKind, run.UpdatedAt.Format("2006-01-02 15:04:05"))
		if run.LastError != "" {
			line += "  error=" + run.LastError
		}
		fmt.Fprintln(w, line)
	}
	return nil
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
