package loop

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var testCfg = config{
	DefaultBranch:   "main",
	AgentMention:    "@codex",
	TokenSecret:     "LOOP_TOKEN",
	CIWorkflow:      "CI",
	CIWorkflowsYAML: `["CI"]`,
	TagPrefix:       "v",
	ReleaseCron:     "0 23 * * *",
}

var testCrons = map[string]string{"planner": "0 * * * *", "builder": "30 * * * *", "coherence": "0 7 * * *"}

// renderable is every template a full scaffold touches, with the per-role config
// applied the same way scaffold does.
func renderCases() map[string]config {
	cases := map[string]config{
		"templates/loop-triage.yml.tmpl":      testCfg,
		"templates/loop-release.yml.tmpl":     testCfg,
		"templates/prompts/triage.md.tmpl":    testCfg,
		"templates/prompts/planner.md.tmpl":   testCfg,
		"templates/prompts/builder.md.tmpl":   testCfg,
		"templates/prompts/coherence.md.tmpl": testCfg,
		"templates/prompts/security.md.tmpl":  testCfg,
	}
	for role, d := range dispatchRoles {
		rc := testCfg
		rc.Role, rc.WorkflowName, rc.IssueTitle, rc.Group, rc.Cron = role, d.workflowName, d.issueTitle, d.group, d.defaultCron
		cases["dispatch:"+role] = rc
	}
	return cases
}

func TestRenderIsPlaceholderFreeAndKeepsGHAExpressions(t *testing.T) {
	for name, cfg := range renderCases() {
		tmplName := name
		if strings.HasPrefix(name, "dispatch:") {
			tmplName = "templates/dispatch.yml.tmpl"
		}
		rendered, err := render(tmplName, cfg)
		if err != nil {
			t.Fatalf("render %s: %v", name, err)
		}
		s := string(rendered)

		// No unresolved substitution delimiters remain in any template.
		if strings.Contains(s, "<<") || strings.Contains(s, ">>") {
			t.Errorf("%s still contains << >> placeholders", name)
		}
	}
}

func TestBaseBranchSubstitutedIntoPrompts(t *testing.T) {
	// The base branch appears in the PR-opening instructions of these prompts.
	for _, p := range []string{"planner", "builder", "coherence", "security"} {
		s := mustRender(t, "templates/prompts/"+p+".md.tmpl", testCfg)
		if !strings.Contains(s, "--base main") {
			t.Errorf("%s prompt missing substituted base branch", p)
		}
	}
}

func TestWorkflowTemplatesPreserveGHAAndAreStructural(t *testing.T) {
	// Only the workflow YAML templates (not the markdown prompts).
	wf := map[string]config{
		"templates/loop-triage.yml.tmpl":  testCfg,
		"templates/loop-release.yml.tmpl": testCfg,
	}
	for role, d := range dispatchRoles {
		rc := testCfg
		rc.Role, rc.WorkflowName, rc.IssueTitle, rc.Group, rc.Cron = role, d.workflowName, d.issueTitle, d.group, d.defaultCron
		wf["dispatch:"+role] = rc
	}
	for name, cfg := range wf {
		tmplName := name
		if strings.HasPrefix(name, "dispatch:") {
			tmplName = "templates/dispatch.yml.tmpl"
		}
		s := mustRender(t, tmplName, cfg)
		if !strings.Contains(s, "${{ secrets.LOOP_TOKEN") {
			t.Errorf("%s lost its ${{ secrets.LOOP_TOKEN }} expression", name)
		}
		for _, key := range []string{"name:", "on:", "jobs:"} {
			if !strings.Contains(s, key) {
				t.Errorf("%s missing top-level %q", name, key)
			}
		}
	}
}

func TestDispatchWorkflowsStripPromptComments(t *testing.T) {
	// The posted body must not include the prompt's editorial <!-- --> header;
	// the workflow strips it. Guard the sed directive in both dispatch paths.
	rc := testCfg
	d := dispatchRoles["planner"]
	rc.Role, rc.WorkflowName, rc.IssueTitle, rc.Group, rc.Cron = "planner", d.workflowName, d.issueTitle, d.group, d.defaultCron
	for _, tc := range []struct {
		name, tmpl string
		cfg        config
	}{
		{"dispatch", "templates/dispatch.yml.tmpl", rc},
		{"triage", "templates/loop-triage.yml.tmpl", testCfg},
	} {
		s := mustRender(t, tc.tmpl, tc.cfg)
		if !strings.Contains(s, `/<!--/,/-->/d`) {
			t.Errorf("%s workflow does not strip prompt HTML comments before posting", tc.name)
		}
	}
}

func TestPromptsLeaveRuntimeTokensLiteral(t *testing.T) {
	// __ISSUE__ must survive render (the workflow substitutes it at runtime).
	for _, p := range []string{"planner", "builder", "coherence", "triage", "security"} {
		s := mustRender(t, "templates/prompts/"+p+".md.tmpl", testCfg)
		if !strings.Contains(s, "__ISSUE__") {
			t.Errorf("%s prompt lost its __ISSUE__ runtime token", p)
		}
	}
	// triage additionally uses __RUNURL__.
	if s := mustRender(t, "templates/prompts/triage.md.tmpl", testCfg); !strings.Contains(s, "__RUNURL__") {
		t.Error("triage prompt lost its __RUNURL__ runtime token")
	}
}

func TestScaffoldAllRolesWritesEverything(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, wfDir, "ci.yml"), "name: CI\n")

	roles := []string{"planner", "builder", "triage", "coherence", "security", "release"}
	if err := scaffold(dir, testCfg, roles, testCrons, false); err != nil {
		t.Fatalf("scaffold: %v", err)
	}

	wantWorkflows := []string{"loop-planner.yml", "loop-builder.yml", "loop-triage.yml", "loop-coherence.yml", "loop-security.yml", "loop-release.yml"}
	for _, w := range wantWorkflows {
		if !fileExists(filepath.Join(dir, wfDir, w)) {
			t.Errorf("expected %s", w)
		}
	}
	// Dispatch + triage roles have prompts; release does not.
	for _, p := range []string{"planner.md", "builder.md", "triage.md", "coherence.md", "security.md"} {
		if !fileExists(filepath.Join(dir, promptDir, p)) {
			t.Errorf("expected prompt %s", p)
		}
	}
	if fileExists(filepath.Join(dir, promptDir, "release.md")) {
		t.Error("release should not have a prompt")
	}
	if _, missing := verifyState(dir); len(missing) != 0 {
		t.Errorf("verify reported missing after full scaffold: %v", missing)
	}
}

func TestScaffoldDefaultRolesOmitsOptional(t *testing.T) {
	dir := t.TempDir()
	if err := scaffold(dir, testCfg, []string{"planner", "builder", "triage"}, testCrons, false); err != nil {
		t.Fatalf("scaffold: %v", err)
	}
	if fileExists(filepath.Join(dir, wfDir, "loop-coherence.yml")) {
		t.Error("coherence should not be written by default")
	}
	if fileExists(filepath.Join(dir, wfDir, "loop-release.yml")) {
		t.Error("release should not be written by default")
	}
}

func TestReinitForceKeepsPromptsRefreshesWorkflows(t *testing.T) {
	dir := t.TempDir()
	roles := []string{"planner", "builder", "triage"}
	if err := scaffold(dir, testCfg, roles, testCrons, false); err != nil {
		t.Fatalf("scaffold: %v", err)
	}

	// Customize a prompt and edit direction/queue, as a real user would.
	customPrompt := filepath.Join(dir, promptDir, "builder.md")
	mustWrite(t, customPrompt, "MY CUSTOM BUILDER POLICY")
	northStar := filepath.Join(dir, loopDir, "NORTH_STAR.md")
	mustWrite(t, northStar, "MY MISSION")

	// Re-run with --force to refresh workflow mechanics.
	if err := scaffold(dir, testCfg, roles, testCrons, true); err != nil {
		t.Fatalf("re-scaffold --force: %v", err)
	}

	// Policy (prompt, North Star) must survive --force untouched.
	if b, _ := os.ReadFile(customPrompt); string(b) != "MY CUSTOM BUILDER POLICY" {
		t.Errorf("--force clobbered a customized prompt: %q", b)
	}
	if b, _ := os.ReadFile(northStar); string(b) != "MY MISSION" {
		t.Errorf("--force clobbered the North Star: %q", b)
	}
	// Mechanism (workflow) must be regenerated (present and non-empty).
	if b, _ := os.ReadFile(filepath.Join(dir, wfDir, "loop-builder.yml")); !strings.Contains(string(b), "Loop: Builder") {
		t.Error("--force did not refresh the workflow")
	}
}

func TestCIWorkflowListRendersAsYAMLArray(t *testing.T) {
	if got := yamlStringArray([]string{"Harness (E2E)", "Lint", "Run Tests"}); got != `["Harness (E2E)", "Lint", "Run Tests"]` {
		t.Errorf("yamlStringArray = %q", got)
	}
	if got := splitCSV("Harness (E2E), Lint ,Run Tests"); strings.Join(got, "|") != "Harness (E2E)|Lint|Run Tests" {
		t.Errorf("splitCSV = %v", got)
	}
	if got := splitCSV("  "); strings.Join(got, "|") != "CI" {
		t.Errorf("splitCSV empty should default to CI, got %v", got)
	}
	// The triage workflow must embed the array so workflow_run watches all of them.
	cfg := testCfg
	cfg.CIWorkflowsYAML = `["Harness (E2E)", "Lint", "Run Tests"]`
	s := mustRender(t, "templates/loop-triage.yml.tmpl", cfg)
	if !strings.Contains(s, `workflows: ["Harness (E2E)", "Lint", "Run Tests"]`) {
		t.Errorf("triage workflow does not watch the CI workflow list:\n%s", s)
	}
}

func TestParseRoles(t *testing.T) {
	if got, err := parseRoles("all"); err != nil || len(got) != len(allRoles) {
		t.Errorf("all => %v, %v", got, err)
	}
	// Canonical order preserved regardless of input order.
	got, err := parseRoles("release,planner")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(got, ",") != "planner,release" {
		t.Errorf("expected canonical order planner,release; got %v", got)
	}
	if _, err := parseRoles("bogus"); err == nil {
		t.Error("expected error for unknown role")
	}
	if _, err := parseRoles(""); err == nil {
		t.Error("expected error for empty roles")
	}
}

func TestVerifyMissingPromptFails(t *testing.T) {
	dir := t.TempDir()
	if err := scaffold(dir, testCfg, []string{"planner", "builder", "triage"}, testCrons, false); err != nil {
		t.Fatalf("scaffold: %v", err)
	}
	// Delete a prompt → verify must flag it.
	if err := os.Remove(filepath.Join(dir, promptDir, "builder.md")); err != nil {
		t.Fatal(err)
	}
	_, missing := verifyState(dir)
	found := false
	for _, m := range missing {
		if strings.Contains(m, "builder.md") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected verify to flag the missing builder prompt; got %v", missing)
	}
}

func mustRender(t *testing.T, tmplName string, cfg config) string {
	t.Helper()
	b, err := render(tmplName, cfg)
	if err != nil {
		t.Fatalf("render %s: %v", tmplName, err)
	}
	return string(b)
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
