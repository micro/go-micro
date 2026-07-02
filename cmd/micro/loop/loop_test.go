package loop

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderProducesValidPlaceholderFreeYAML(t *testing.T) {
	cfg := config{
		DefaultBranch: "main",
		AgentMention:  "@codex",
		TokenSecret:   "LOOP_TOKEN",
		CIWorkflow:    "CI",
		PlannerCron:   "0 * * * *",
		BuilderCron:   "30 * * * *",
	}

	for tmplName := range workflows {
		rendered, err := render(tmplName, cfg)
		if err != nil {
			t.Fatalf("render %s: %v", tmplName, err)
		}
		s := string(rendered)

		// No unresolved substitution delimiters should remain...
		if strings.Contains(s, "<<") || strings.Contains(s, ">>") {
			t.Errorf("%s still contains << >> placeholders after render", tmplName)
		}
		// ...but GitHub Actions' own ${{ }} expressions must survive verbatim.
		if !strings.Contains(s, "${{ secrets.LOOP_TOKEN") {
			t.Errorf("%s lost its ${{ secrets.LOOP_TOKEN }} expression", tmplName)
		}
		// The configured values must be substituted in.
		if !strings.Contains(s, "@codex") {
			t.Errorf("%s missing agent mention", tmplName)
		}
		// Structural sanity: a workflow needs these top-level keys.
		for _, key := range []string{"name:", "on:", "jobs:"} {
			if !strings.Contains(s, key) {
				t.Errorf("%s missing top-level %q", tmplName, key)
			}
		}
	}
}

func TestInitThenVerify(t *testing.T) {
	dir := t.TempDir()
	// A non-loop workflow so verify's CI-gate check passes.
	mustWrite(t, filepath.Join(dir, ".github/workflows/ci.yml"), "name: CI\n")

	if err := scaffold(dir, config{
		DefaultBranch: "main",
		AgentMention:  "@codex",
		TokenSecret:   "LOOP_TOKEN",
		CIWorkflow:    "CI",
		PlannerCron:   "0 * * * *",
		BuilderCron:   "30 * * * *",
	}, false); err != nil {
		t.Fatalf("scaffold: %v", err)
	}

	for _, dest := range workflows {
		if !fileExists(filepath.Join(dir, dest)) {
			t.Errorf("expected %s to be written", dest)
		}
	}
	for _, dest := range docs {
		if !fileExists(filepath.Join(dir, dest)) {
			t.Errorf("expected %s to be written", dest)
		}
	}

	// A second scaffold without --force must fail on an existing workflow.
	if err := scaffold(dir, config{DefaultBranch: "main", AgentMention: "@codex", TokenSecret: "LOOP_TOKEN", CIWorkflow: "CI", PlannerCron: "0 * * * *", BuilderCron: "30 * * * *"}, false); err == nil {
		t.Error("expected second scaffold without --force to fail")
	}
}

func TestVerifyMissingFilesFails(t *testing.T) {
	dir := t.TempDir()
	if _, missing := verifyState(dir); len(missing) == 0 {
		t.Error("expected missing files in an empty dir")
	}
}

func TestVerifyWarnsWithoutCIGate(t *testing.T) {
	dir := t.TempDir()
	if err := scaffold(dir, config{DefaultBranch: "main", AgentMention: "@codex", TokenSecret: "LOOP_TOKEN", CIWorkflow: "CI", PlannerCron: "0 * * * *", BuilderCron: "30 * * * *"}, false); err != nil {
		t.Fatalf("scaffold: %v", err)
	}
	// Only loop-* workflows exist → no CI gate.
	if hasCIWorkflow(dir) {
		t.Error("expected no CI gate when only loop-* workflows are present")
	}
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
