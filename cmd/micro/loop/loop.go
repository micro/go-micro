// Package loop implements the 'micro loop' command, which scaffolds and
// verifies an autonomous improvement loop for a repository.
//
// The loop is a set of GitHub Actions workflows — a planner that keeps a ranked
// queue, a builder that builds the top item as a single-concern PR, and a triage
// pass that turns CI failures into fix issues — that dispatch a coding agent by
// @mention on a fresh tracking issue each run. `micro loop init` writes those
// workflows (plus a NORTH_STAR and PRIORITIES queue) into a repo; `micro loop
// verify` checks that a repo is wired correctly.
package loop

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/urfave/cli/v2"
	"go-micro.dev/v6/cmd"
)

//go:embed templates/*
var templatesFS embed.FS

// config is the substitution surface for the workflow templates. It is the
// whole "config vs core" boundary: the workflows are the reusable core, these
// fields are what a given repo tunes.
type config struct {
	DefaultBranch string // base branch for the loop's PRs (e.g. main)
	AgentMention  string // how the workflows summon the agent (e.g. @codex)
	TokenSecret   string // repo secret holding the user PAT that drives dispatch
	CIWorkflow    string // name: of the CI workflow triage watches for failures
	PlannerCron   string // cron for the planner
	BuilderCron   string // cron for the builder
}

// generated workflow files: template name -> destination (relative to repo root).
var workflows = map[string]string{
	"templates/loop-planner.yml.tmpl": ".github/workflows/loop-planner.yml",
	"templates/loop-builder.yml.tmpl": ".github/workflows/loop-builder.yml",
	"templates/loop-triage.yml.tmpl":  ".github/workflows/loop-triage.yml",
}

// static (non-templated) docs: template name -> destination.
var docs = map[string]string{
	"templates/NORTH_STAR.md": ".github/loop/NORTH_STAR.md",
	"templates/PRIORITIES.md": ".github/loop/PRIORITIES.md",
}

func init() {
	cmd.Register(&cli.Command{
		Name:  "loop",
		Usage: "Scaffold an autonomous improvement loop for a repository",
		Description: `Set up a self-improving loop for a repo: GitHub Actions workflows that
dispatch a coding agent to plan, build, and triage — gated by CI.

The loop has three roles:
  planner  keeps a ranked queue in .github/loop/PRIORITIES.md
  builder  builds the top open item as a single-concern PR (auto-merged on green CI)
  triage   turns CI failures into scoped fix issues back into the queue

Direction lives in .github/loop/NORTH_STAR.md — edit it to steer the loop.

Examples:
  # Scaffold the loop into the current repo
  micro loop init

  # Customize the agent, token secret, base branch, and CI workflow name
  micro loop init --agent @codex --token-secret LOOP_TOKEN \
    --branch main --ci-workflow CI

  # Check that a repo is wired correctly
  micro loop verify`,
		Subcommands: []*cli.Command{
			{
				Name:  "init",
				Usage: "Scaffold the loop workflows and queue into a repo",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "dir", Usage: "Target repo directory", Value: "."},
					&cli.StringFlag{Name: "branch", Usage: "Base branch for the loop's PRs (auto-detected if empty)"},
					&cli.StringFlag{Name: "agent", Usage: "How the workflows summon the agent (an @mention)", Value: "@codex"},
					&cli.StringFlag{Name: "token-secret", Usage: "Repo secret holding the user PAT that drives dispatch", Value: "LOOP_TOKEN"},
					&cli.StringFlag{Name: "ci-workflow", Usage: "name: of the CI workflow triage watches for failures", Value: "CI"},
					&cli.StringFlag{Name: "planner-cron", Usage: "Cron schedule for the planner", Value: "0 * * * *"},
					&cli.StringFlag{Name: "builder-cron", Usage: "Cron schedule for the builder", Value: "30 * * * *"},
					&cli.BoolFlag{Name: "force", Usage: "Overwrite existing loop files"},
				},
				Action: runInit,
			},
			{
				Name:   "verify",
				Usage:  "Verify a repo is wired for the loop",
				Flags:  []cli.Flag{&cli.StringFlag{Name: "dir", Usage: "Target repo directory", Value: "."}},
				Action: runVerify,
			},
		},
	})
}

func runInit(c *cli.Context) error {
	dir := c.String("dir")
	cfg := config{
		DefaultBranch: c.String("branch"),
		AgentMention:  strings.TrimSpace(c.String("agent")),
		TokenSecret:   strings.TrimSpace(c.String("token-secret")),
		CIWorkflow:    c.String("ci-workflow"),
		PlannerCron:   c.String("planner-cron"),
		BuilderCron:   c.String("builder-cron"),
	}
	if cfg.DefaultBranch == "" {
		cfg.DefaultBranch = detectDefaultBranch(dir)
	}
	if !strings.HasPrefix(cfg.AgentMention, "@") {
		cfg.AgentMention = "@" + cfg.AgentMention
	}

	if err := scaffold(dir, cfg, c.Bool("force")); err != nil {
		return err
	}
	printNextSteps(cfg)
	return nil
}

// scaffold renders the workflow templates and writes the loop files into dir.
// Static docs (NORTH_STAR, PRIORITIES) are never clobbered even with force, so
// re-running init can't wipe curated direction or a hand-tuned queue.
func scaffold(dir string, cfg config, force bool) error {
	for tmplName, dest := range workflows {
		rendered, err := render(tmplName, cfg)
		if err != nil {
			return err
		}
		if err := writeFile(filepath.Join(dir, dest), rendered, force); err != nil {
			return err
		}
		fmt.Printf("  wrote %s\n", dest)
	}

	for tmplName, dest := range docs {
		full := filepath.Join(dir, dest)
		if fileExists(full) {
			fmt.Printf("  kept  %s (already exists)\n", dest)
			continue
		}
		b, err := templatesFS.ReadFile(tmplName)
		if err != nil {
			return err
		}
		if err := writeFile(full, b, true); err != nil {
			return err
		}
		fmt.Printf("  wrote %s\n", dest)
	}
	return nil
}

// verifyState reports what's wrong with dir's loop setup: warnings are
// non-fatal, missing are required files that aren't present.
func verifyState(dir string) (warnings, missing []string) {
	for _, dest := range workflows {
		if !fileExists(filepath.Join(dir, dest)) {
			missing = append(missing, dest)
		}
	}
	for _, dest := range docs {
		if !fileExists(filepath.Join(dir, dest)) {
			missing = append(missing, dest)
		}
	}
	// The loop is only as good as its gate: warn if there's no non-loop
	// workflow to serve as CI.
	if !hasCIWorkflow(dir) {
		warnings = append(warnings, "no non-loop workflow found in .github/workflows — the loop needs a CI gate (build/test/lint) to merge safely")
	}
	return warnings, missing
}

func runVerify(c *cli.Context) error {
	dir := c.String("dir")
	warnings, missing := verifyState(dir)

	for _, m := range missing {
		fmt.Printf("  MISSING  %s\n", m)
	}
	for _, w := range warnings {
		fmt.Printf("  WARN     %s\n", w)
	}

	if len(missing) > 0 {
		return fmt.Errorf("loop is not fully scaffolded (%d file(s) missing) — run `micro loop init`", len(missing))
	}

	fmt.Println("  OK       loop workflows and queue are present")
	fmt.Println()
	fmt.Println("Reminders the CLI can't check:")
	fmt.Println("  • The token secret must be set in the repo (Settings → Secrets).")
	fmt.Println("  • Branch protection must require the CI checks with 0 approvals,")
	fmt.Println("    so the builder's auto-merge can land PRs on green CI.")
	if len(warnings) > 0 {
		return fmt.Errorf("%d warning(s) — see above", len(warnings))
	}
	return nil
}

func render(tmplName string, cfg config) ([]byte, error) {
	b, err := templatesFS.ReadFile(tmplName)
	if err != nil {
		return nil, err
	}
	// Custom delimiters so GitHub Actions' own ${{ }} expressions pass through
	// untouched — only << >> placeholders are substituted.
	t, err := template.New(filepath.Base(tmplName)).Delims("<<", ">>").Option("missingkey=error").Parse(string(b))
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", tmplName, err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, cfg); err != nil {
		return nil, fmt.Errorf("render %s: %w", tmplName, err)
	}
	return buf.Bytes(), nil
}

func writeFile(path string, content []byte, force bool) error {
	if fileExists(path) && !force {
		return fmt.Errorf("%s already exists (use --force to overwrite)", path)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, content, 0o644)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// hasCIWorkflow reports whether .github/workflows holds any workflow that is
// not one of the loop's own (i.e. a plausible CI gate).
func hasCIWorkflow(dir string) bool {
	entries, err := os.ReadDir(filepath.Join(dir, ".github", "workflows"))
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, "loop-") {
			continue
		}
		if strings.HasSuffix(name, ".yml") || strings.HasSuffix(name, ".yaml") {
			return true
		}
	}
	return false
}

// detectDefaultBranch best-effort resolves the repo's default branch, falling
// back to "main".
func detectDefaultBranch(dir string) string {
	out, err := exec.Command("git", "-C", dir, "symbolic-ref", "--short", "refs/remotes/origin/HEAD").Output()
	if err == nil {
		ref := strings.TrimSpace(string(out))
		if i := strings.LastIndex(ref, "/"); i >= 0 {
			ref = ref[i+1:]
		}
		if ref != "" {
			return ref
		}
	}
	return "main"
}

func printNextSteps(cfg config) {
	fmt.Printf(`
Loop scaffolded. Next steps (the CLI can't do these for you):

  1. Edit .github/loop/NORTH_STAR.md — the direction the loop aligns to.
     Seed .github/loop/PRIORITIES.md with a few real items.

  2. Add a repo secret named %s: a fine-grained user PAT (contents + pull
     requests + issues write) for an account the agent (%s) responds to.
     The workflows no-op until this secret exists.

  3. Ensure a CI workflow named %q exists and that branch protection on %q
     requires its checks with 0 approving reviews — that green-CI gate is
     what lets the builder auto-merge safely.

  4. Commit these files, then trigger a run:
     Actions → "Loop: Planner" / "Loop: Builder" → Run workflow.

Verify anytime with: micro loop verify
`, cfg.TokenSecret, cfg.AgentMention, cfg.CIWorkflow, cfg.DefaultBranch)
}
