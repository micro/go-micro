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
	if !subcommands["agent"]["demo"] {
		t.Fatal("first-agent walkthrough missing no-secret boundary: agent demo")
	}
	if !subcommands["agent"]["doctor"] {
		t.Fatal("first-agent walkthrough missing recovery boundary: agent doctor")
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
	if demoIdx, guideIdx := strings.Index(out.String(), "micro agent demo"), strings.Index(out.String(), "no-secret-first-agent.html"); demoIdx < 0 || guideIdx < 0 || demoIdx > guideIdx {
		t.Fatalf("micro docs should lead with micro agent demo before guide links:\n%s", out.String())
	}
	for _, want := range []string{
		"micro agent demo",
		"no-secret-first-agent.html",
		"your-first-agent.html",
		"debugging-agents.html",
		"zero-to-hero.html",
		"micro agent preflight  # before micro run: prerequisites",
		"micro run",
		"micro chat",
		"micro agent doctor     # after micro run: chat/gateway/inspect recovery",
		"micro inspect agent <name>",
		"micro agent history <name>",
	} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("micro docs output missing %q:\n%s", want, out.String())
		}
	}
	if strings.Contains(out.String(), "micro runs") {
		t.Fatalf("micro docs output should use the first-agent inspect command, not the legacy runs shortcut:\n%s", out.String())
	}

	agent := commandByName(t, "agent")
	if !strings.Contains(agent.Usage, "micro agent demo") {
		t.Fatalf("micro agent help should advertise the no-secret demo; usage was %q", agent.Usage)
	}
	doctor := subcommandByName(t, agent, "doctor")
	for _, want := range []string{"chat", "gateway", "registration", "provider", "inspect", "after micro run"} {
		if !strings.Contains(doctor.Usage, want) {
			t.Fatalf("micro agent doctor usage should advertise after-run recovery for %q; usage was %q", want, doctor.Usage)
		}
	}

	demo := subcommandByName(t, agent, "demo")
	out.Reset()
	if err := demo.Action(cli.NewContext(app, nil, nil)); err != nil {
		t.Fatalf("micro agent demo failed: %v", err)
	}
	for _, want := range []string{
		"No-secret first-agent demo",
		"go test ./internal/harness/zero-to-hero-ci -run TestNoSecretFirstAgentTranscript -count=1",
		"provider-free",
		"micro agent preflight  # before micro run: prerequisites",
		"micro chat",
		"micro agent doctor     # after micro run: chat/gateway/inspect recovery",
		"micro inspect agent <name>",
		"your-first-agent.html",
		"debugging-agents.html",
		"zero-to-hero.html",
	} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("micro agent demo output missing %q:\n%s", want, out.String())
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

func subcommandByName(t *testing.T, command *cli.Command, name string) *cli.Command {
	t.Helper()
	for _, subcommand := range command.Subcommands {
		if subcommand.Name == name {
			return subcommand
		}
	}
	t.Fatalf("missing subcommand %q under %q", name, command.Name)
	return nil
}
