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
		"go test ./cmd/micro -run TestFirstAgentWalkthroughCLIBoundaries -count=1",
		"go test ./cmd/micro -run TestZeroToHeroCLIBoundaries -count=1",
		"go test ./cmd/micro/cli/deploy -run TestDeployDryRun -count=1",
		"go test ./examples/support -run 'TestRunSupportMockSmoke|TestZeroToHeroReadmeDocumentsLifecycle' -count=1",
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
		"/docs/guides/no-secret-first-agent.html",
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

func TestFirstAgentWayfindingDocs(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", ".."))
	checks := []struct {
		name    string
		file    string
		heading string
		links   []string
	}{
		{
			name:    "README first-agent on-ramp",
			file:    filepath.Join(root, "README.md"),
			heading: "### First agent on-ramp",
			links: []string{
				"internal/website/docs/guides/no-secret-first-agent.md",
				"internal/website/docs/guides/your-first-agent.md",
				"internal/website/docs/guides/debugging-agents.md",
				"internal/website/docs/guides/zero-to-hero.md",
			},
		},
		{
			name:    "website getting-started on-ramp",
			file:    filepath.Join(root, "internal", "website", "docs", "getting-started.md"),
			heading: "### First-agent on-ramp",
			links: []string{
				"guides/no-secret-first-agent.html",
				"guides/your-first-agent.html",
				"guides/debugging-agents.html",
				"guides/zero-to-hero.html",
			},
		},
	}

	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			doc := firstMarkdownSection(t, readFile(t, check.file), check.heading)
			last := -1
			for _, link := range check.links {
				idx := strings.Index(doc, link)
				if idx == -1 {
					t.Fatalf("%s missing first-agent wayfinding link %q; keep the no-secret → first-agent → debugging → 0→hero path discoverable", check.name, link)
				}
				if idx < last {
					t.Fatalf("%s link %q appeared out of order; expected no-secret → first-agent → debugging → 0→hero", check.name, link)
				}
				last = idx
			}
		})
	}
}

func TestNoSecretFirstAgentTranscript(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", ".."))
	guide := readFile(t, filepath.Join(root, "internal", "website", "docs", "guides", "no-secret-first-agent.md"))

	for _, want := range []string{
		"go run ./examples/support",
		"go test ./examples/support -run TestRunSupportMockSmoke -count=1",
		"make harness",
		"micro agent preflight",
		"micro run",
		"micro chat assistant",
		"micro inspect agent assistant",
		"go test ./cmd/micro -run TestFirstAgentWalkthroughCLIBoundaries -count=1",
		"No-secret first-agent transcript",
	} {
		if !strings.Contains(guide, want) {
			t.Fatalf("no-secret first-agent transcript missing %q", want)
		}
	}

	readme := readFile(t, filepath.Join(root, "README.md"))
	if !strings.Contains(readme, "internal/website/docs/guides/no-secret-first-agent.md") {
		t.Fatal("README does not point to the no-secret first-agent transcript")
	}

	firstAgent := readFile(t, filepath.Join(root, "internal", "website", "docs", "guides", "your-first-agent.md"))
	if !strings.Contains(firstAgent, "no-secret-first-agent.html") {
		t.Fatal("Your First Agent guide does not point to the no-secret transcript")
	}
}

func firstMarkdownSection(t *testing.T, doc, heading string) string {
	t.Helper()
	start := strings.Index(doc, heading)
	if start == -1 {
		t.Fatalf("missing %q section", heading)
	}
	section := doc[start+len(heading):]
	if next := strings.Index(section, "\n##"); next != -1 {
		section = section[:next]
	}
	return section
}

func readFile(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(name)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(data)
}
