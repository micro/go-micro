package zerotoheroci

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestZeroToHeroReferenceDocs(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", ".."))

	guide := readFile(t, filepath.Join(root, "internal", "website", "docs", "guides", "zero-to-hero.md"))
	for _, want := range []string{
		"make harness",
		"go test ./cmd/micro/cli/new -run TestZeroToOne -count=1",
		"go test ./cmd/micro -run TestZeroToHeroCLIBoundaries -count=1",
		"go test ./cmd/micro/cli/deploy -run TestDeployDryRun -count=1",
		"./internal/harness/zero-to-hero-ci/run.sh",
		"go run ./internal/harness/agent-flow",
		"make provider-conformance-mock",
		"internal/harness/plan-delegate",
		"internal/harness/universe",
	} {
		if !strings.Contains(guide, want) {
			t.Fatalf("0→hero guide missing %q", want)
		}
	}

	readme := readFile(t, filepath.Join(root, "README.md"))
	if !strings.Contains(readme, "internal/website/docs/guides/zero-to-hero.md") {
		t.Fatal("README does not point to the canonical 0→hero guide")
	}

	nav := readFile(t, filepath.Join(root, "internal", "website", "_data", "navigation.yml"))
	if !strings.Contains(nav, "0→hero Reference") || !strings.Contains(nav, "/docs/guides/zero-to-hero.html") {
		t.Fatal("website navigation does not expose the canonical 0→hero guide")
	}
}

func TestGuidesNavigationLeadsWithDoing(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", ".."))
	nav := readFile(t, filepath.Join(root, "internal", "website", "_data", "navigation.yml"))

	orderedGuides := []string{
		"/docs/guides/your-first-agent.html",
		"/docs/guides/zero-to-hero.html",
		"/docs/guides/plan-delegate.html",
		"/docs/guides/agent-guardrails.html",
		"/docs/guides/agents-and-workflows.html",
		"/docs/guides/agent-patterns.html",
		"/docs/guides/agent-harness.html",
		"/docs/guides/agent-loops.html",
	}

	last := -1
	for _, guide := range orderedGuides {
		idx := strings.Index(nav, guide)
		if idx == -1 {
			t.Fatalf("website navigation does not expose %s", guide)
		}
		if idx < last {
			t.Fatalf("website navigation should lead with hands-on guides; %s appeared out of order", guide)
		}
		last = idx

		doc := strings.TrimPrefix(strings.TrimSuffix(guide, ".html"), "/docs/") + ".md"
		if _, err := os.Stat(filepath.Join(root, "internal", "website", "docs", doc)); err != nil {
			t.Fatalf("navigation links to missing guide %s: %v", guide, err)
		}
	}
}

func readFile(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(name)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(data)
}
