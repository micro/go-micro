// Package a2a provides the 'micro a2a' command for the Agent2Agent gateway.
package a2a

import (
	"fmt"
	"strings"

	"github.com/urfave/cli/v2"
	"go-micro.dev/v6/cmd"
	"go-micro.dev/v6/gateway/a2a"
	"go-micro.dev/v6/registry"
)

func init() {
	cmd.Register(&cli.Command{
		Name:  "a2a",
		Usage: "Agent2Agent (A2A) protocol gateway",
		Description: `Expose registered agents over the A2A protocol so other agents can
discover and call them.

Examples:
  # Serve the A2A gateway over HTTP
  micro a2a serve --address :4000 --base_url https://agents.example.com

  # List agents and their A2A card URLs
  micro a2a list

Agents are discovered from the registry (the ones advertising type=agent);
an Agent Card is generated for each from its registry metadata, and
incoming A2A tasks are translated to the agent's Agent.Chat RPC. This is
the agent-side analog of 'micro mcp', which exposes services as tools.`,
		Subcommands: []*cli.Command{
			{
				Name:  "serve",
				Usage: "Start the A2A gateway",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "address", Usage: "Address to listen on", Value: ":4000"},
					&cli.StringFlag{Name: "base_url", Usage: "Public base URL for Agent Cards"},
				},
				Action: func(c *cli.Context) error {
					return a2a.Serve(a2a.Options{
						Registry: registry.DefaultRegistry,
						Address:  c.String("address"),
						BaseURL:  c.String("base_url"),
					})
				},
			},
			{
				Name:   "list",
				Usage:  "List agents and their A2A card URLs",
				Flags:  []cli.Flag{&cli.StringFlag{Name: "base_url", Usage: "Public base URL for Agent Cards", Value: "http://localhost:4000"}},
				Action: listAgents,
			},
		},
	})
}

func listAgents(c *cli.Context) error {
	base := strings.TrimRight(c.String("base_url"), "/")
	svcs, err := registry.ListServices()
	if err != nil {
		return err
	}
	found := false
	for _, s := range svcs {
		recs, err := registry.GetService(s.Name)
		if err != nil || len(recs) == 0 {
			continue
		}
		meta := recs[0].Metadata
		isAgent := meta != nil && meta["type"] == "agent"
		if !isAgent && len(recs[0].Nodes) > 0 {
			nm := recs[0].Nodes[0].Metadata
			isAgent = nm != nil && nm["type"] == "agent"
		}
		if !isAgent {
			continue
		}
		found = true
		fmt.Printf("  \033[36m◆\033[0m %-20s %s/agents/%s\n", s.Name, base, s.Name)
	}
	if !found {
		fmt.Println("  No agents registered.")
	}
	return nil
}
