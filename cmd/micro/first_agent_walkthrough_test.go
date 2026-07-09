package main

import (
	"bytes"
	"os"
	"path/filepath"
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

	for _, want := range []string{"new", "run", "chat", "inspect", "agent", "docs", "examples"} {
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
	if !strings.Contains(chat.Description, "services") || !strings.Contains(chat.Description, "agent") || !strings.Contains(chat.Description, `micro chat assistant --prompt`) {
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

	examples := commandByName(t, "examples")
	if !strings.Contains(examples.Usage, "first-agent") {
		t.Fatalf("micro examples should advertise the first-agent examples path; usage was %q", examples.Usage)
	}
	out.Reset()
	if err := examples.Action(cli.NewContext(app, nil, nil)); err != nil {
		t.Fatalf("micro examples failed: %v", err)
	}
	for _, want := range []string{
		"First-agent examples",
		"go run ./examples/first-agent",
		"go test ./internal/harness/zero-to-hero-ci -run TestNoSecretFirstAgentTranscript -count=1",
		"go run ./examples/support",
		"micro agent demo",
		"micro docs",
		"micro zero-to-hero",
		"no-secret-first-agent.html",
		"your-first-agent.html",
		"debugging-agents.html",
		"zero-to-hero.html",
	} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("micro examples output missing %q:\n%s", want, out.String())
		}
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

func TestFirstAgentDocsMatchCLIOutput(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	outputs := map[string]string{
		"micro docs":         commandOutput(t, commandByName(t, "docs")),
		"micro examples":     commandOutput(t, commandByName(t, "examples")),
		"micro zero-to-hero": commandOutput(t, commandByName(t, "zero-to-hero")),
	}
	agent := commandByName(t, "agent")
	outputs["micro agent demo"] = commandOutput(t, subcommandByName(t, agent, "demo"))

	contracts := []struct {
		name    string
		file    string
		markers []string
	}{
		{
			name: "README first-agent on-ramp",
			file: filepath.Join(root, "README.md"),
			markers: []string{
				"micro agent demo",
				"micro examples",
				"micro zero-to-hero",
				"examples/first-agent/",
				"examples/support/",
				"internal/website/docs/guides/no-secret-first-agent.md",
				"internal/website/docs/guides/your-first-agent.md",
				"internal/website/docs/guides/debugging-agents.md",
				"internal/website/docs/guides/zero-to-hero.md",
			},
		},
		{
			name: "website getting-started first-agent on-ramp",
			file: filepath.Join(root, "internal", "website", "docs", "getting-started.md"),
			markers: []string{
				"micro agent demo",
				"micro examples",
				"micro zero-to-hero",
				"github.com/micro/go-micro/tree/master/examples/first-agent",
				"github.com/micro/go-micro/tree/master/examples/support",
				"guides/no-secret-first-agent.html",
				"guides/your-first-agent.html",
				"guides/debugging-agents.html",
				"guides/zero-to-hero.html",
			},
		},
	}

	for _, contract := range contracts {
		doc := readTestFile(t, contract.file)
		for _, marker := range contract.markers {
			if !strings.Contains(doc, marker) {
				t.Fatalf("%s missing documented first-agent marker %q", contract.name, marker)
			}
			if isCLIContractMarker(marker) && !cliOutputsContain(outputs, marker) {
				t.Fatalf("%s documents %q, but none of the first-agent CLI outputs mention it; keep README/website breadcrumbs aligned with micro agent demo/examples/zero-to-hero", contract.name, marker)
			}
			assertMaintainedFirstAgentPath(t, root, marker)
		}
	}
}

func commandOutput(t *testing.T, command *cli.Command) string {
	t.Helper()
	var out bytes.Buffer
	app := cli.NewApp()
	app.Writer = &out
	if err := command.Action(cli.NewContext(app, nil, nil)); err != nil {
		t.Fatalf("%s failed: %v", command.Name, err)
	}
	return out.String()
}

func cliOutputsContain(outputs map[string]string, marker string) bool {
	for command, out := range outputs {
		if command == marker || strings.Contains(out, marker) {
			return true
		}
	}
	return false
}

func isCLIContractMarker(marker string) bool {
	return strings.HasPrefix(marker, "micro ") || strings.HasPrefix(marker, "go run ") || strings.HasPrefix(marker, "go test ") || strings.Contains(marker, ".html")
}

func assertMaintainedFirstAgentPath(t *testing.T, root, marker string) {
	t.Helper()
	pathChecks := map[string]string{
		"go run ./examples/first-agent":                              "examples/first-agent",
		"examples/first-agent/":                                      "examples/first-agent",
		"examples/support/":                                          "examples/support",
		"internal/website/docs/guides/no-secret-first-agent.md":      "internal/website/docs/guides/no-secret-first-agent.md",
		"internal/website/docs/guides/your-first-agent.md":           "internal/website/docs/guides/your-first-agent.md",
		"internal/website/docs/guides/debugging-agents.md":           "internal/website/docs/guides/debugging-agents.md",
		"internal/website/docs/guides/zero-to-hero.md":               "internal/website/docs/guides/zero-to-hero.md",
		"guides/no-secret-first-agent.html":                          "internal/website/docs/guides/no-secret-first-agent.md",
		"guides/your-first-agent.html":                               "internal/website/docs/guides/your-first-agent.md",
		"guides/debugging-agents.html":                               "internal/website/docs/guides/debugging-agents.md",
		"guides/zero-to-hero.html":                                   "internal/website/docs/guides/zero-to-hero.md",
		"github.com/micro/go-micro/tree/master/examples/first-agent": "examples/first-agent",
		"github.com/micro/go-micro/tree/master/examples/support":     "examples/support",
	}
	path, ok := pathChecks[marker]
	if !ok {
		return
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(path))); err != nil {
		t.Fatalf("documented first-agent path %q from marker %q does not resolve: %v", path, marker, err)
	}
}

func readTestFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
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
