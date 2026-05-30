// Package resource provides CLI commands that map directly onto
// go-micro's core interfaces — registry, broker, store, and config.
//
// Each interface gets its own top-level command with verbs that mirror
// the interface methods, so the framework's building blocks are
// inspectable and manipulable from the terminal:
//
//	micro registry list
//	micro broker publish <topic> <message>
//	micro store read <key>
//	micro config get <key>
//
// New resource commands are registered by appending to the commands
// slice in init — see registry.go, broker.go, store.go, config.go for
// the per-interface implementations.
package resource

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
	"go-micro.dev/v5/cmd"
)

// commandFunc returns a cli.Command for a single core interface. Add a
// new one here to expose another package on the CLI.
var commandFuncs = []func() *cli.Command{
	registryCommand,
	brokerCommand,
	storeCommand,
	configCommand,
}

func init() {
	for _, fn := range commandFuncs {
		cmd.Register(fn())
	}
}

// printJSON writes v as indented JSON to stdout.
func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// fail returns a cli error with a consistent prefix.
func fail(format string, args ...any) error {
	return cli.Exit(fmt.Sprintf(format, args...), 1)
}
