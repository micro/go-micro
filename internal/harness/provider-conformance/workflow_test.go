package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHarnessWorkflowSchedulesLiveProviderMatrix(t *testing.T) {
	path := filepath.Join(repoRoot(), ".github", "workflows", "harness.yml")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read harness workflow: %v", err)
	}
	workflow := string(b)

	checks := []string{
		`name: Harness (E2E)`,
		`schedule:`,
		`cron: "17 * * * *"`,
		`workflow_dispatch:`,
		`harness-live:`,
		`if: github.event_name == 'schedule' || github.event_name == 'workflow_dispatch'`,
		`ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}`,
		`OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}`,
		`GEMINI_API_KEY: ${{ secrets.GEMINI_API_KEY }}`,
		`GROQ_API_KEY: ${{ secrets.GROQ_API_KEY }}`,
		`MINIMAX_API_KEY: ${{ secrets.MINIMAX_API_KEY }}`,
		`MISTRAL_API_KEY: ${{ secrets.MISTRAL_API_KEY }}`,
		`TOGETHER_API_KEY: ${{ secrets.TOGETHER_API_KEY }}`,
		`ATLASCLOUD_API_KEY: ${{ secrets.ATLASCLOUD_API_KEY }}`,
		`-summary-json provider-conformance-summary.json`,
		`-summary-markdown provider-conformance-summary.md`,
		`-capabilities-markdown provider-capabilities.md`,
		`actions/upload-artifact@v4`,
	}
	for _, want := range checks {
		if !strings.Contains(workflow, want) {
			t.Fatalf("harness workflow missing %q", want)
		}
	}
}
