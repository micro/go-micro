// Package loop implements the 'micro loop' command, which scaffolds and
// verifies an autonomous improvement loop for a repository.
//
// The loop is a set of GitHub Actions workflows that dispatch a coding agent by
// @mention on a fresh tracking issue each run. It has up to five roles:
//
//	planner    keeps a ranked queue in .github/loop/PRIORITIES.md
//	builder    builds the top open item as a single-concern PR (auto-merged on green CI)
//	triage     turns CI failures into scoped fix issues back into the queue
//	coherence  keeps README/docs/CHANGELOG aligned with the North Star (opt-in)
//	release    cuts the next patch tag when the branch has new commits (opt-in)
//
// The workflows are the MECHANISM; each dispatch role's instruction lives in an
// editable .github/loop/prompts/<role>.md file — the POLICY. That split is what
// lets any repo (including go-micro itself) customize behavior by editing prompt
// files rather than forking the CLI. `micro loop init` writes it all; `micro
// loop verify` checks the wiring.
package loop

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/urfave/cli/v2"
	"go-micro.dev/v6/cmd"
)

//go:embed templates/*
var templatesFS embed.FS

// config is the substitution surface for the templates — the whole config-vs-core
// boundary. The workflows and prompts are the reusable core; these are what a
// given repo tunes.
type config struct {
	// Shared.
	DefaultBranch string // base branch for the loop's PRs (e.g. main)
	AgentMention  string // how the workflows summon the agent (e.g. @codex)
	TokenSecret   string // repo secret holding the user PAT that drives dispatch
	CIWorkflow    string // name: of the CI workflow triage watches for failures

	// Per-dispatch-role (set while rendering each one).
	Role         string
	WorkflowName string
	IssueTitle   string
	Group        string
	Cron         string

	// Release role.
	TagPrefix   string // tag prefix to match/bump, e.g. "v"
	ReleaseCron string
}

// dispatchRole is a cron-driven role rendered from templates/dispatch.yml.tmpl.
type dispatchRole struct {
	workflowName string
	issueTitle   string
	group        string
	cronFlag     string
	defaultCron  string
}

var dispatchRoles = map[string]dispatchRole{
	"planner":   {"Loop: Planner", "Loop: planning review", "loop-planner", "planner-cron", "0 * * * *"},
	"builder":   {"Loop: Builder", "Loop: build increment", "loop-builder", "builder-cron", "30 * * * *"},
	"coherence": {"Loop: Coherence", "Loop: coherence review", "loop-coherence", "coherence-cron", "0 7 * * *"},
}

// allRoles is the full set, in a stable order, for --roles=all and help text.
var allRoles = []string{"planner", "builder", "triage", "coherence", "release"}

const (
	promptDir = ".github/loop/prompts"
	loopDir   = ".github/loop"
	wfDir     = ".github/workflows"
)

func init() {
	cmd.Register(&cli.Command{
		Name:  "loop",
		Usage: "Scaffold an autonomous improvement loop for a repository",
		Description: `Set up a self-improving loop for a repo: GitHub Actions workflows that
dispatch a coding agent to plan, build, triage, and (optionally) keep docs
coherent and cut releases — gated by CI.

Roles (choose with --roles, default: planner,builder,triage):
  planner    keeps a ranked queue in .github/loop/PRIORITIES.md
  builder    builds the top open item as a single-concern PR (auto-merged on green CI)
  triage     turns CI failures into scoped fix issues back into the queue
  coherence  keeps README/docs/CHANGELOG aligned with the North Star
  release    cuts the next patch tag when the branch has new commits

Each dispatch role's instruction is an editable file in .github/loop/prompts/ —
edit those to steer behavior. Direction lives in .github/loop/NORTH_STAR.md.

Examples:
  # Scaffold the default loop (planner, builder, triage)
  micro loop init

  # The full loop, all five roles
  micro loop init --roles all

  # Customize the agent, token secret, base branch, and CI workflow name
  micro loop init --agent @codex --token-secret LOOP_TOKEN \
    --branch main --ci-workflow CI

  # Check that a repo is wired correctly
  micro loop verify`,
		Subcommands: []*cli.Command{
			{
				Name:  "init",
				Usage: "Scaffold the loop workflows, prompts, and queue into a repo",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "dir", Usage: "Target repo directory", Value: "."},
					&cli.StringFlag{Name: "roles", Usage: "Comma-separated roles, or 'all'", Value: "planner,builder,triage"},
					&cli.StringFlag{Name: "branch", Usage: "Base branch for the loop's PRs (auto-detected if empty)"},
					&cli.StringFlag{Name: "agent", Usage: "How the workflows summon the agent (an @mention)", Value: "@codex"},
					&cli.StringFlag{Name: "token-secret", Usage: "Repo secret holding the user PAT that drives dispatch", Value: "LOOP_TOKEN"},
					&cli.StringFlag{Name: "ci-workflow", Usage: "name: of the CI workflow triage watches for failures", Value: "CI"},
					&cli.StringFlag{Name: "planner-cron", Usage: "Cron schedule for the planner", Value: "0 * * * *"},
					&cli.StringFlag{Name: "builder-cron", Usage: "Cron schedule for the builder", Value: "30 * * * *"},
					&cli.StringFlag{Name: "coherence-cron", Usage: "Cron schedule for the coherence role", Value: "0 7 * * *"},
					&cli.StringFlag{Name: "release-cron", Usage: "Cron schedule for the release role", Value: "0 23 * * *"},
					&cli.StringFlag{Name: "tag-prefix", Usage: "Tag prefix the release role matches and bumps", Value: "v"},
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
	roles, err := parseRoles(c.String("roles"))
	if err != nil {
		return err
	}

	cfg := config{
		DefaultBranch: c.String("branch"),
		AgentMention:  strings.TrimSpace(c.String("agent")),
		TokenSecret:   strings.TrimSpace(c.String("token-secret")),
		CIWorkflow:    c.String("ci-workflow"),
		TagPrefix:     c.String("tag-prefix"),
		ReleaseCron:   c.String("release-cron"),
	}
	if cfg.DefaultBranch == "" {
		cfg.DefaultBranch = detectDefaultBranch(dir)
	}
	if !strings.HasPrefix(cfg.AgentMention, "@") {
		cfg.AgentMention = "@" + cfg.AgentMention
	}

	crons := map[string]string{
		"planner":   c.String("planner-cron"),
		"builder":   c.String("builder-cron"),
		"coherence": c.String("coherence-cron"),
	}

	if err := scaffold(dir, cfg, roles, crons, c.Bool("force")); err != nil {
		return err
	}
	printNextSteps(cfg, roles)
	return nil
}

// parseRoles resolves the --roles flag into a validated, stable-ordered set.
func parseRoles(spec string) ([]string, error) {
	if strings.TrimSpace(spec) == "all" {
		return append([]string(nil), allRoles...), nil
	}
	want := map[string]bool{}
	for _, r := range strings.Split(spec, ",") {
		r = strings.TrimSpace(r)
		if r == "" {
			continue
		}
		if !isRole(r) {
			return nil, fmt.Errorf("unknown role %q (valid: %s, or 'all')", r, strings.Join(allRoles, ", "))
		}
		want[r] = true
	}
	if len(want) == 0 {
		return nil, fmt.Errorf("no roles selected")
	}
	var out []string
	for _, r := range allRoles { // preserve canonical order
		if want[r] {
			out = append(out, r)
		}
	}
	return out, nil
}

func isRole(r string) bool {
	for _, x := range allRoles {
		if x == r {
			return true
		}
	}
	return false
}

// scaffold renders the selected roles into dir. The split is deliberate:
//   - Workflows are the MECHANISM — regenerated, and overwritten with --force.
//   - Prompts, NORTH_STAR, and PRIORITIES are the POLICY — written once and
//     never clobbered, even with --force, so re-running init to refresh the
//     workflow mechanics can't wipe curated instructions, direction, or queue.
func scaffold(dir string, cfg config, roles []string, crons map[string]string, force bool) error {
	for _, role := range roles {
		switch role {
		case "triage":
			if err := renderTo(dir, "templates/loop-triage.yml.tmpl", filepath.Join(wfDir, "loop-triage.yml"), cfg, force); err != nil {
				return err
			}
			if err := renderKeep(dir, "templates/prompts/triage.md.tmpl", filepath.Join(promptDir, "triage.md"), cfg); err != nil {
				return err
			}
		case "release":
			if err := renderTo(dir, "templates/loop-release.yml.tmpl", filepath.Join(wfDir, "loop-release.yml"), cfg, force); err != nil {
				return err
			}
		default: // dispatch roles
			d := dispatchRoles[role]
			rc := cfg
			rc.Role = role
			rc.WorkflowName = d.workflowName
			rc.IssueTitle = d.issueTitle
			rc.Group = d.group
			rc.Cron = crons[role]
			if rc.Cron == "" {
				rc.Cron = d.defaultCron
			}
			if err := renderTo(dir, "templates/dispatch.yml.tmpl", filepath.Join(wfDir, "loop-"+role+".yml"), rc, force); err != nil {
				return err
			}
			if err := renderKeep(dir, "templates/prompts/"+role+".md.tmpl", filepath.Join(promptDir, role+".md"), cfg); err != nil {
				return err
			}
		}
	}

	// Direction + queue: policy, written once, never clobbered.
	if err := renderKeep(dir, "templates/NORTH_STAR.md", filepath.Join(loopDir, "NORTH_STAR.md"), cfg); err != nil {
		return err
	}
	return renderKeep(dir, "templates/PRIORITIES.md", filepath.Join(loopDir, "PRIORITIES.md"), cfg)
}

// renderTo renders a template with cfg and writes it to dir/dest (honoring force).
func renderTo(dir, tmplName, dest string, cfg config, force bool) error {
	rendered, err := render(tmplName, cfg)
	if err != nil {
		return err
	}
	if err := writeFile(filepath.Join(dir, dest), rendered, force); err != nil {
		return err
	}
	fmt.Printf("  wrote %s\n", dest)
	return nil
}

// renderKeep writes dir/dest only if it does not already exist — used for
// policy files (prompts, North Star, queue) so re-running init never clobbers
// customizations, regardless of --force.
func renderKeep(dir, tmplName, dest string, cfg config) error {
	full := filepath.Join(dir, dest)
	if fileExists(full) {
		fmt.Printf("  kept  %s (already exists)\n", dest)
		return nil
	}
	rendered, err := render(tmplName, cfg)
	if err != nil {
		return err
	}
	if err := writeFile(full, rendered, true); err != nil {
		return err
	}
	fmt.Printf("  wrote %s\n", dest)
	return nil
}

// verifyState reports what's wrong with dir's loop setup: warnings are
// non-fatal, missing are required files that aren't present.
func verifyState(dir string) (warnings, missing []string) {
	// A loop needs direction, a queue, and at least one role workflow.
	for _, dest := range []string{filepath.Join(loopDir, "NORTH_STAR.md"), filepath.Join(loopDir, "PRIORITIES.md")} {
		if !fileExists(filepath.Join(dir, dest)) {
			missing = append(missing, dest)
		}
	}

	present := presentLoopWorkflows(dir)
	if len(present) == 0 {
		missing = append(missing, wfDir+"/loop-*.yml (no role workflows found)")
	}

	// Every dispatch/triage role workflow needs its prompt file. (release has none.)
	for _, role := range present {
		if role == "release" {
			continue
		}
		prompt := filepath.Join(promptDir, role+".md")
		if !fileExists(filepath.Join(dir, prompt)) {
			missing = append(missing, prompt+" (prompt for the loop-"+role+" workflow)")
		}
	}

	// The loop is only as good as its gate.
	if !hasCIWorkflow(dir) {
		warnings = append(warnings, "no non-loop workflow found in "+wfDir+" — the loop needs a CI gate (build/test/lint) to merge safely")
	}
	return warnings, missing
}

// presentLoopWorkflows returns the role names for which a loop-<role>.yml exists.
func presentLoopWorkflows(dir string) []string {
	entries, err := os.ReadDir(filepath.Join(dir, wfDir))
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		name := e.Name()
		if !strings.HasPrefix(name, "loop-") {
			continue
		}
		role := strings.TrimSuffix(strings.TrimSuffix(strings.TrimPrefix(name, "loop-"), ".yml"), ".yaml")
		out = append(out, role)
	}
	sort.Strings(out)
	return out
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
		return fmt.Errorf("loop is not fully scaffolded (%d item(s) missing) — run `micro loop init`", len(missing))
	}

	fmt.Printf("  OK       loop is wired: %s\n", strings.Join(presentLoopWorkflows(dir), ", "))
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
	entries, err := os.ReadDir(filepath.Join(dir, wfDir))
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

func printNextSteps(cfg config, roles []string) {
	fmt.Printf(`
Loop scaffolded (%s). Next steps (the CLI can't do these for you):

  1. Edit .github/loop/NORTH_STAR.md — the direction the loop aligns to.
     Seed .github/loop/PRIORITIES.md with a few real items.
     Tune the per-role instructions in .github/loop/prompts/ if you like.

  2. Add a repo secret named %s: a fine-grained user PAT (contents + pull
     requests + issues write) for an account the agent (%s) responds to.
     The workflows no-op until this secret exists.

  3. Ensure a CI workflow named %q exists and that branch protection on %q
     requires its checks with 0 approving reviews — that green-CI gate is
     what lets the builder auto-merge safely.

  4. Commit these files, then trigger a run from the Actions tab.

Verify anytime with: micro loop verify
`, strings.Join(roles, ", "), cfg.TokenSecret, cfg.AgentMention, cfg.CIWorkflow, cfg.DefaultBranch)
}
