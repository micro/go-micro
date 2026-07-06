package main

import (
	"bytes"
	"strings"
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

	for _, want := range []string{"run", "chat", "flow", "inspect", "deploy", "zero-to-hero"} {
		if !commands[want] {
			t.Fatalf("missing %q command", want)
		}
	}
	if !subcommands["flow"]["runs"] {
		t.Fatal("missing inspect boundary: flow runs")
	}
	if !subcommands["inspect"]["agent"] || !subcommands["inspect"]["flow"] {
		t.Fatal("missing inspect boundary: inspect agent/flow")
	}

	var hasDeployDryRun bool
	for _, command := range microcmd.DefaultCmd.App().Commands {
		if command.Name != "deploy" {
			continue
		}
		for _, flag := range command.Flags {
			for _, name := range flag.Names() {
				if name == "dry-run" {
					hasDeployDryRun = true
				}
			}
		}
	}
	if !hasDeployDryRun {
		t.Fatal("missing deploy boundary: deploy --dry-run")
	}
}

func TestZeroToHeroCommandPrintsMaintainedNoSecretPath(t *testing.T) {
	app := microcmd.DefaultCmd.App()
	var out bytes.Buffer
	oldWriter := app.Writer
	app.Writer = &out
	t.Cleanup(func() { app.Writer = oldWriter })

	if err := app.Run([]string{"micro", "zero-to-hero"}); err != nil {
		t.Fatalf("micro zero-to-hero failed: %v", err)
	}

	got := out.String()
	for _, want := range []string{
		"0→hero no-secret lifecycle demo",
		"./internal/harness/zero-to-hero-ci/run.sh",
		"go run ./examples/first-agent",
		"go run ./examples/support",
		"make harness",
		"services → agents → workflows",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("micro zero-to-hero output missing %q:\n%s", want, got)
		}
	}
}
