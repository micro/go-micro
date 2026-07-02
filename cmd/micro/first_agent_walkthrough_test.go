package main

import (
	"strings"
	"testing"

	"github.com/urfave/cli/v2"
	microcmd "go-micro.dev/v6/cmd"
)

func TestFirstAgentWalkthroughCLIBoundaries(t *testing.T) {
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

	for _, want := range []string{"new", "run", "chat", "inspect", "agent"} {
		if !commands[want] {
			t.Fatalf("first-agent walkthrough missing %q command", want)
		}
	}
	if !subcommands["agent"]["preflight"] {
		t.Fatal("first-agent walkthrough missing preflight boundary: agent preflight")
	}
	if !subcommands["inspect"]["agent"] {
		t.Fatal("first-agent walkthrough missing inspect boundary: inspect agent")
	}

	chat := commandByName(t, "chat")
	if !strings.Contains(chat.Description, "services") || !strings.Contains(chat.Description, "agent") {
		t.Fatalf("micro chat should describe the service-to-agent walkthrough boundary; description was %q", chat.Description)
	}
}

func commandByName(t *testing.T, name string) *cli.Command {
	t.Helper()
	for _, command := range microcmd.DefaultCmd.App().Commands {
		if command.Name == name {
			return command
		}
	}
	t.Fatalf("missing command %q", name)
	return nil
}
