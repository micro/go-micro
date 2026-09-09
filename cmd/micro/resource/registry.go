package resource

import (
	"fmt"

	"github.com/urfave/cli/v2"
	"go-micro.dev/v6/registry"
)

// registryCommand exposes the registry interface: list, get, watch.
func registryCommand() *cli.Command {
	return &cli.Command{
		Name:  "registry",
		Usage: "Inspect the service registry",
		Description: `Interact with the service registry.

  micro registry list            List all registered services
  micro registry get <name>      Show nodes and endpoints for a service
  micro registry watch           Stream registration events`,
		Subcommands: []*cli.Command{
			{
				Name:   "list",
				Usage:  "List all registered services",
				Action: registryList,
			},
			{
				Name:      "get",
				Usage:     "Show details for a service",
				ArgsUsage: "<name>",
				Action:    registryGet,
			},
			{
				Name:   "watch",
				Usage:  "Stream registration events",
				Action: registryWatch,
			},
		},
	}
}

func registryList(c *cli.Context) error {
	services, err := registry.ListServices()
	if err != nil {
		return fail("list services: %v", err)
	}
	out := make([]map[string]any, 0, len(services))
	for _, s := range services {
		out = append(out, map[string]any{
			"name":    s.Name,
			"version": s.Version,
		})
	}
	return printJSON(out)
}

func registryGet(c *cli.Context) error {
	name := c.Args().First()
	if name == "" {
		return fail("usage: micro registry get <name>")
	}
	services, err := registry.GetService(name)
	if err != nil {
		return fail("get service %q: %v", name, err)
	}
	if len(services) == 0 {
		return fail("service %q not found", name)
	}
	return printJSON(services)
}

func registryWatch(c *cli.Context) error {
	w, err := registry.Watch()
	if err != nil {
		return fail("watch registry: %v", err)
	}
	defer w.Stop()

	fmt.Println("Watching registry for changes (Ctrl-C to stop)...")
	for {
		res, err := w.Next()
		if err != nil {
			return fail("watch: %v", err)
		}
		name := ""
		version := ""
		if res.Service != nil {
			name = res.Service.Name
			version = res.Service.Version
		}
		fmt.Printf("%-10s %s %s\n", res.Action, name, version)
	}
}
