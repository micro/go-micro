// Package agent registers the 'micro agent' CLI commands.
package agent

import (
	"encoding/json"
	"fmt"

	"github.com/urfave/cli/v2"
	goagent "go-micro.dev/v5/agent"
	"go-micro.dev/v5/cmd"
	"go-micro.dev/v5/registry"
	"go-micro.dev/v5/store"
)

func init() {
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
				Usage:     "Show an agent's stored conversation history",
				ArgsUsage: "[name]",
				Action: func(c *cli.Context) error {
					name := c.Args().First()
					if name == "" {
						return fmt.Errorf("usage: micro agent history [name]")
					}
					// Read from the agent's scoped state store (database
					// "agent", table = name) — available whether or not the
					// agent is currently running.
					mem := goagent.NewMemory(store.Scope(store.DefaultStore, "agent", name), "history", 1000)
					msgs := mem.Messages()
					if len(msgs) == 0 {
						fmt.Printf("  No history for agent %q.\n", name)
						return nil
					}
					for _, m := range msgs {
						fmt.Printf("  \033[2m%s:\033[0m %v\n", m.Role, m.Content)
					}
					return nil
				},
			},
		},
	})
}
