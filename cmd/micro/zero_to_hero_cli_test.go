package main

import (
	"testing"

	microcmd "go-micro.dev/v6/cmd"
)

func TestZeroToHeroCLIBoundaries(t *testing.T) {
	commands := map[string]bool{}
	subcommands := map[string]map[string]bool{}
	for _, command := range microcmd.DefaultCmd.App().Commands {
		commands[command.Name] = true
		for _, subcommand := range command.Subcommands {
			if subcommands[command.Name] == nil {
				subcommands[command.Name] = map[string]bool{}
			}
			subcommands[command.Name][subcommand.Name] = true
		}
	}

	for _, want := range []string{"run", "chat", "flow"} {
		if !commands[want] {
			t.Fatalf("missing %q command", want)
		}
	}
	if !subcommands["flow"]["runs"] {
		t.Fatal("missing inspect boundary: flow runs")
	}
}
