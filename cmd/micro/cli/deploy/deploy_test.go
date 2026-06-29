package deploy

import (
	"flag"
	"os"
	"strings"
	"testing"

	"github.com/urfave/cli/v2"
	"go-micro.dev/v6/cmd/micro/run/config"
)

func newDeployTestContext(t *testing.T, args ...string) *cli.Context {
	t.Helper()
	set := flag.NewFlagSet("deploy", flag.ContinueOnError)
	set.String("path", defaultRemotePath, "")
	set.String("ssh", "", "")
	set.String("service", "", "")
	set.Bool("build", false, "")
	set.Bool("dry-run", false, "")
	if err := set.Parse(args); err != nil {
		t.Fatalf("parse flags: %v", err)
	}
	return cli.NewContext(cli.NewApp(), set, nil)
}

func TestDeployNoTargetExplainsInitAndDeployHandoff(t *testing.T) {
	err := showDeployHelp()
	if err == nil {
		t.Fatal("expected missing target guidance")
	}
	msg := err.Error()
	for _, want := range []string{
		"no deployment target specified",
		"sudo micro init --server",
		"micro deploy user@your-server",
		"deploy prod",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing %q in guidance:\n%s", want, msg)
		}
	}
}

func TestDeployListsConfiguredTargetsWhenNoTargetProvided(t *testing.T) {
	err := showDeployTargets(&config.Config{Deploy: map[string]*config.DeployTarget{
		"prod":    {Name: "prod", SSH: "deploy@prod.example.com"},
		"staging": {Name: "staging", SSH: "deploy@staging.example.com"},
	}})
	if err == nil {
		t.Fatal("expected configured target guidance")
	}
	msg := err.Error()
	for _, want := range []string{
		"Available deploy targets:",
		"prod -> deploy@prod.example.com",
		"staging -> deploy@staging.example.com",
		"micro deploy <target>",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing %q in configured target guidance:\n%s", want, msg)
		}
	}
}

func TestResolveDeployTargetUsesConfigTargetAndPath(t *testing.T) {
	ctx := newDeployTestContext(t, "prod")
	cfg := &config.Config{Deploy: map[string]*config.DeployTarget{
		"prod": {Name: "prod", SSH: "deploy@prod.example.com", Path: "/srv/micro"},
	}}

	target, remotePath := resolveDeployTarget(ctx, ctx.Args().First(), cfg)
	if target != "deploy@prod.example.com" {
		t.Fatalf("target = %q, want configured SSH", target)
	}
	if remotePath != "/srv/micro" {
		t.Fatalf("remotePath = %q, want configured path", remotePath)
	}
}

func TestResolveDeployTargetAllowsCLIPathOverride(t *testing.T) {
	ctx := newDeployTestContext(t, "--path", "/tmp/micro", "prod")
	cfg := &config.Config{Deploy: map[string]*config.DeployTarget{
		"prod": {Name: "prod", SSH: "deploy@prod.example.com", Path: "/srv/micro"},
	}}

	target, remotePath := resolveDeployTarget(ctx, ctx.Args().First(), cfg)
	if target != "deploy@prod.example.com" {
		t.Fatalf("target = %q, want configured SSH", target)
	}
	if remotePath != "/tmp/micro" {
		t.Fatalf("remotePath = %q, want CLI override", remotePath)
	}
}

func TestDeployConfigParserSupportsDeployTargets(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/micro.mu"
	content := `service api
    path ./api

deploy prod
    ssh deploy@prod.example.com
    path /srv/micro
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := config.ParseMu(path)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	prod := cfg.Deploy["prod"]
	if prod == nil {
		t.Fatal("missing prod deploy target")
	}
	if prod.SSH != "deploy@prod.example.com" || prod.Path != "/srv/micro" {
		t.Fatalf("deploy target = %#v", prod)
	}
}

func TestDeployDryRunPlansConfiguredTargetWithoutRemoteSideEffects(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(dir+"/micro.mu", []byte(`service api
    path ./api

deploy prod
    ssh deploy@prod.example.com
    path /srv/micro
`), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldwd); err != nil {
			t.Errorf("restore cwd: %v", err)
		}
	})

	ctx := newDeployTestContext(t, "--dry-run", "prod")
	if err := Deploy(ctx); err != nil {
		t.Fatalf("dry-run deploy: %v", err)
	}
}

func TestDeployDryRunValidatesRequestedService(t *testing.T) {
	ctx := newDeployTestContext(t, "--dry-run", "--service", "missing", "prod")
	cfg := &config.Config{Services: map[string]*config.Service{
		"api": {Name: "api", Path: "./api"},
	}}

	err := printDeployPlan(ctx, "deploy@prod.example.com", cfg, defaultRemotePath)
	if err == nil {
		t.Fatal("expected dry-run to validate service names")
	}
	if !strings.Contains(err.Error(), "service 'missing' not found in configuration") {
		t.Fatalf("unexpected error: %v", err)
	}
}
