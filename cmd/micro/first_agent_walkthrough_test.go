package main

import (
	"bytes"
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

	for _, want := range []string{"new", "run", "chat", "inspect", "agent", "docs"} {
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

	docs := commandByName(t, "docs")
	if !strings.Contains(docs.Usage, "first-agent") || !strings.Contains(docs.Usage, "0→hero") {
		t.Fatalf("micro docs should advertise the first-agent and 0→hero docs path; usage was %q", docs.Usage)
	}
	var out bytes.Buffer
	app := cli.NewApp()
	app.Writer = &out
	if err := docs.Action(cli.NewContext(app, nil, nil)); err != nil {
		t.Fatalf("micro docs failed: %v", err)
	}
	for _, want := range []string{
		"no-secret-first-agent.html",
		"your-first-agent.html",
		"debugging-agents.html",
		"zero-to-hero.html",
		"micro agent preflight",
		"micro run",
		"micro chat",
		"micro inspect agent",
	} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("micro docs output missing %q:\n%s", want, out.String())
		}
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
