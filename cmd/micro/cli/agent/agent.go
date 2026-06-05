// Package agent registers the 'micro agent' CLI commands.
package agent

import (
	"encoding/json"
	"fmt"

	"github.com/urfave/cli/v2"
	"go-micro.dev/v5/cmd"
	"go-micro.dev/v5/registry"
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
							continue
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
		},
	})
}
