package zerotoheroci

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	goagent "go-micro.dev/v6/agent"
	"go-micro.dev/v6/store"
)

func TestZeroToHeroReferenceDocs(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", ".."))

	guide := readFile(t, filepath.Join(root, "internal", "website", "docs", "guides", "zero-to-hero.md"))
	for _, want := range []string{
		"make harness",
		"make inner-loop",
		"go test ./cmd/micro/cli/new -run TestZeroToOne -count=1",
		"go test ./cmd/micro -run TestFirstAgentWalkthroughCLIBoundaries -count=1",
		"go test ./cmd/micro -run TestZeroToHeroCLIBoundaries -count=1",
		"go test ./cmd/micro/cli/deploy -run TestDeployDryRun -count=1",
		"go test ./examples/first-agent -run TestRunFirstAgent -count=1",
		"go test ./examples/support -run 'TestRunSupportMockSmoke|TestZeroToHeroReadmeDocumentsLifecycle|TestZeroToHeroInspectTranscript' -count=1",
		"./internal/harness/zero-to-hero-ci/run.sh",
		"micro zero-to-hero",
		"go run ./internal/harness/agent-flow",
		"make provider-conformance-mock",
		"internal/harness/plan-delegate",
		"internal/harness/universe",
	} {
		if !strings.Contains(guide, want) {
			t.Fatalf("0→hero guide missing %q", want)
		}
	}

	runScript := readFile(t, filepath.Join(root, "internal", "harness", "zero-to-hero-ci", "run.sh"))
	for _, want := range []string{
		"go test ./cmd/micro/cli/new -run TestZeroToOne -count=1",
		"go test ./cmd/micro -run 'TestFirstAgentWalkthroughCLIBoundaries|TestExamplesWayfindingIndexStaysLinked|TestExamplesCommandPointsAtWayfindingIndex|TestZeroToHeroCLIBoundaries|TestZeroToHeroCommandPrintsMaintainedNoSecretPath' -count=1",
		"go test ./cmd/micro/cli/deploy -run TestDeployDryRun -count=1",
		"go test ./examples/first-agent -run TestRunFirstAgent -count=1",
		"go test ./examples/support -run 'TestRunSupportMockSmoke|TestZeroToHeroReadmeDocumentsLifecycle|TestZeroToHeroInspectTranscript' -count=1",
	} {
		if !strings.Contains(runScript, want) {
			t.Fatalf("0→hero CI run script missing lifecycle command %q", want)
		}
	}
	for _, want := range []string{
		"scaffold:",
		"run/chat/inspect:",
		"deploy dry-run:",
		"chat/inspect:",
		"first-agent app:",
		"0→hero app:",
		"flow history:",
	} {
		if !strings.Contains(runScript, want) {
			t.Fatalf("0→hero CI run script missing debuggable boundary label %q", want)
		}
	}

	readme := readFile(t, filepath.Join(root, "README.md"))
	if !strings.Contains(readme, "internal/website/docs/guides/zero-to-hero.md") {
		t.Fatal("README does not point to the canonical 0→hero guide")
	}
	if !strings.Contains(readme, "make inner-loop") {
		t.Fatal("README does not expose the focused CLI inner-loop contract")
	}

	nav := readFile(t, filepath.Join(root, "internal", "website", "_data", "navigation.yml"))
	if !strings.Contains(nav, "0→hero Reference") || !strings.Contains(nav, "/docs/guides/zero-to-hero.html") {
		t.Fatal("website navigation does not expose the canonical 0→hero guide")
	}
}

func TestZeroToHeroDeployDryRunCommandSmoke(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", ".."))
	absRoot, err := filepath.Abs(root)
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}

	bin := filepath.Join(t.TempDir(), "micro")
	build := exec.Command("go", "build", "-o", bin, "./cmd/micro")
	build.Dir = absRoot
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build micro CLI for deploy dry-run smoke: %v\n%s", err, out)
	}

	workspace := t.TempDir()
	writeFile(t, filepath.Join(workspace, "micro.mu"), `service api
    path ./api

deploy prod
    ssh deploy@prod.example.com
    path /srv/micro
`)
	if err := os.Mkdir(filepath.Join(workspace, "api"), 0o755); err != nil {
		t.Fatalf("create service dir: %v", err)
	}

	cmd := exec.Command(bin, "deploy", "--dry-run", "prod")
	cmd.Dir = workspace
	cmd.Env = append(os.Environ(), "MICRO_CONFIG_FILE="+filepath.Join(workspace, "micro.mu"))
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("documented deploy dry-run command failed: %v\n%s", err, out)
	}

	got := string(out)
	for _, want := range []string{
		"micro deploy --dry-run",
		"Target",
		"deploy@prod.example.com",
		"Remote path",
		"/srv/micro",
		"Services",
		"api",
		"No SSH, rsync, systemd, or remote deployment was performed.",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("deploy dry-run output missing %q:\n%s", want, got)
		}
	}

	guide := readFile(t, filepath.Join(absRoot, "internal", "website", "docs", "guides", "zero-to-hero.md"))
	if !strings.Contains(guide, "micro deploy --dry-run prod") {
		t.Fatal("0→hero guide must document the same deploy dry-run command covered by CI")
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

func TestYourFirstAgentTutorialSmoke(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", ".."))
	absRoot, err := filepath.Abs(root)
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	guide := readFile(t, filepath.Join(root, "internal", "website", "docs", "guides", "your-first-agent.md"))

	for _, want := range []string{
		"go test ./internal/harness/zero-to-hero-ci -run TestYourFirstAgentTutorialSmoke -count=1",
		"micro agent preflight",
		"mkdir first-agent",
		"go mod init example.com/first-agent",
		"go get go-micro.dev/v6@v6",
		"micro run",
		"micro call task TaskService.Create",
		"micro call task TaskService.List",
		"micro chat assistant",
		"micro inspect agent assistant",
	} {
		if !strings.Contains(guide, want) {
			t.Fatalf("Your First Agent guide missing copy/paste boundary %q", want)
		}
	}

	mainGo := extractFirstAgentMain(t, guide)
	workspace := t.TempDir()
	writeFile(t, filepath.Join(workspace, "go.mod"), "module example.com/first-agent\n\ngo 1.24\n\nrequire go-micro.dev/v6 v6.0.0\n\nreplace go-micro.dev/v6 => "+absRoot+"\n")
	writeFile(t, filepath.Join(workspace, "main.go"), mainGo)

	runInWorkspace(t, workspace, "go", "mod", "tidy")
	runInWorkspace(t, workspace, "go", "test", "./...")
}

func extractFirstAgentMain(t *testing.T, guide string) string {
	t.Helper()
	start := strings.Index(guide, "Add `main.go`:")
	if start == -1 {
		t.Fatal("Your First Agent guide is missing the main.go section")
	}
	rest := guide[start:]
	open := strings.Index(rest, "```go")
	if open == -1 {
		t.Fatal("Your First Agent guide is missing a Go code fence for main.go")
	}
	rest = rest[open+len("```go"):]
	close := strings.Index(rest, "```")
	if close == -1 {
		t.Fatal("Your First Agent guide main.go code fence is not closed")
	}
	return strings.TrimSpace(rest[:close]) + "\n"
}

func writeFile(t *testing.T, name, contents string) {
	t.Helper()
	if err := os.WriteFile(name, []byte(contents), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func runInWorkspace(t *testing.T, workspace, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = workspace
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Your First Agent tutorial command %q does not pass from a clean workspace: %v\n%s", strings.Join(append([]string{name}, args...), " "), err, out)
	}
}

func TestArchitectureDocsAlignWithAgentHarnessLifecycle(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", ".."))
	doc := readFile(t, filepath.Join(root, "internal", "website", "docs", "architecture.md"))

	for _, want := range []string{
		"services → agents → workflows lifecycle",
		"## Service substrate",
		"## Agent harness",
		"## Workflows",
		"## Interop gateways",
		"`model` / `ai.Model`",
		"`store` / memory",
		"`ai.Tools`",
		"`agent`",
		"`flow`",
		"`micro mcp`",
		"`micro a2a`",
		"[AI Integration](ai-integration.html)",
		"[Your First Agent](guides/your-first-agent.html)",
		"[0→hero Reference](guides/zero-to-hero.html)",
	} {
		if !strings.Contains(doc, want) {
			t.Fatalf("architecture doc missing lifecycle marker %q", want)
		}
	}

	assertOrderedMarkers(t, "architecture lifecycle", doc, []string{
		"## Service substrate",
		"## Agent harness",
		"## Workflows",
		"## Interop gateways",
		"## Developer path",
	})
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
				"internal/website/docs/guides/install-troubleshooting.md",
				"micro agent demo",
				"micro examples",
				"micro zero-to-hero",
				"internal/website/docs/guides/no-secret-first-agent.md",
				"internal/website/docs/guides/your-first-agent.md",
				"micro chat",
				"internal/website/docs/guides/debugging-agents.md",
				"micro inspect agent <name>",
				"internal/website/docs/guides/zero-to-hero.md",
			},
		},
		{
			name:    "README examples list",
			file:    filepath.Join(root, "README.md"),
			heading: "## Examples",
			links: []string{
				"examples/README.md",
				"examples/first-agent/",
			},
		},
		{
			name:    "repository examples index",
			file:    filepath.Join(root, "examples", "README.md"),
			heading: "## Recommended first-agent path",
			links: []string{
				"./first-agent/",
				"./support/",
			},
		},
		{
			name:    "repository examples wayfinding index",
			file:    filepath.Join(root, "examples", "INDEX.md"),
			heading: "## Recommended adoption path",
			links: []string{
				"./hello-world/",
				"./first-agent/",
				"./support/",
			},
		},
		{
			name:    "micro README first-agent on-ramp",
			file:    filepath.Join(root, "cmd", "micro", "README.md"),
			heading: "## First agent on-ramp",
			links: []string{
				"micro agent demo",
				"micro examples",
				"micro zero-to-hero",
			},
		},
		{
			name:    "website examples index",
			file:    filepath.Join(root, "internal", "website", "docs", "examples", "index.md"),
			heading: "## Start here",
			links: []string{
				"https://github.com/micro/go-micro/tree/master/examples/first-agent",
				"../guides/no-secret-first-agent.html",
				"../guides/your-first-agent.html",
				"../guides/debugging-agents.html",
				"../guides/zero-to-hero.html",
			},
		},
		{
			name:    "website getting-started on-ramp",
			file:    filepath.Join(root, "internal", "website", "docs", "getting-started.md"),
			heading: "### First-agent on-ramp",
			links: []string{
				"guides/install-troubleshooting.html",
				"micro agent demo",
				"micro examples",
				"micro zero-to-hero",
				"https://github.com/micro/go-micro/blob/master/examples/INDEX.md",
				"https://github.com/micro/go-micro/tree/master/examples/support",
				"https://github.com/micro/go-micro/tree/master/examples/first-agent",
				"guides/no-secret-first-agent.html",
				"guides/your-first-agent.html",
				"micro chat",
				"guides/debugging-agents.html",
				"micro inspect agent <name>",
				"guides/zero-to-hero.html",
			},
		},
		{
			name:    "website quickstart next steps",
			file:    filepath.Join(root, "internal", "website", "docs", "quickstart.md"),
			heading: "## Next Steps",
			links: []string{
				"guides/install-troubleshooting.html",
				"micro agent demo",
				"micro examples",
				"micro zero-to-hero",
				"https://github.com/micro/go-micro/blob/master/examples/INDEX.md",
				"https://github.com/micro/go-micro/tree/master/examples/support",
				"https://github.com/micro/go-micro/tree/master/examples/first-agent",
				"guides/no-secret-first-agent.html",
				"guides/your-first-agent.html",
				"micro chat",
				"guides/debugging-agents.html",
				"micro inspect agent <name>",
				"guides/zero-to-hero.html",
			},
		},
		{
			name:    "website docs index learn more",
			file:    filepath.Join(root, "internal", "website", "docs", "index.md"),
			heading: "## Learn More",
			links: []string{
				"getting-started.html",
				"https://github.com/micro/go-micro/blob/master/examples/INDEX.md",
				"https://github.com/micro/go-micro/tree/master/examples/support",
				"guides/no-secret-first-agent.html",
				"guides/your-first-agent.html",
				"micro chat",
				"guides/debugging-agents.html",
				"micro inspect agent <name>",
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
				assertWayfindingTargetExists(t, root, check.file, link)
				if idx < last {
					t.Fatalf("%s link %q appeared out of order; expected no-secret → first-agent → debugging → 0→hero", check.name, link)
				}
				last = idx
			}
		})
	}
}

func TestFirstAgentWayfindingCanonicalTrailStaysInSync(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", ".."))
	onRampTrail := []string{
		"micro agent demo",
		"micro examples",
		"micro zero-to-hero",
		"examples/INDEX.md",
		"examples/first-agent",
		"examples/support",
		"no-secret-first-agent",
		"your-first-agent",
		"micro chat",
		"debugging-agents",
		"micro inspect agent <name>",
		"zero-to-hero",
	}
	checks := []struct {
		name    string
		file    string
		heading string
		markers []string
	}{
		{
			name:    "README first-agent on-ramp",
			file:    filepath.Join(root, "README.md"),
			heading: "### First agent on-ramp",
			markers: onRampTrail,
		},
		{
			name:    "website docs index first-agent path",
			file:    filepath.Join(root, "internal", "website", "docs", "index.md"),
			heading: "## Learn More",
			markers: onRampTrail,
		},
		{
			name:    "website getting-started first-agent on-ramp",
			file:    filepath.Join(root, "internal", "website", "docs", "getting-started.md"),
			heading: "### First-agent on-ramp",
			markers: onRampTrail,
		},
		{
			name:    "website quickstart next steps",
			file:    filepath.Join(root, "internal", "website", "docs", "quickstart.md"),
			heading: "## Next Steps",
			markers: onRampTrail,
		},
		{
			name:    "examples map recommended adoption path",
			file:    filepath.Join(root, "examples", "INDEX.md"),
			heading: "",
			markers: []string{
				"micro examples",
				"micro zero-to-hero",
				"examples/INDEX.md",
				"examples/first-agent",
				"examples/support",
				"zero-to-hero",
			},
		},
		{
			name:    "no-secret first-agent guide next steps",
			file:    filepath.Join(root, "internal", "website", "docs", "guides", "no-secret-first-agent.md"),
			heading: "",
			markers: []string{
				"micro agent demo",
				"examples/first-agent",
				"examples/support",
				"your-first-agent",
				"micro chat",
				"debugging-agents",
				"micro inspect agent <name>",
			},
		},
		{
			name:    "0→hero guide related examples",
			file:    filepath.Join(root, "internal", "website", "docs", "guides", "zero-to-hero.md"),
			heading: "",
			markers: []string{
				"micro zero-to-hero",
				"examples/first-agent",
				"examples/support",
				"micro inspect agent <name>",
				"zero-to-hero",
			},
		},
	}

	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			section := readFile(t, check.file)
			if check.heading != "" {
				section = firstMarkdownSection(t, section, check.heading)
			}
			for _, marker := range check.markers {
				if !containsWayfindingMarker(section, marker) {
					t.Fatalf("%s missing canonical first-agent wayfinding marker %q; keep README, website, examples, no-secret, and 0→hero surfaces aligned", check.name, marker)
				}
			}
		})
	}
}

func TestFirstAgentWayfindingLinkTargetsResolve(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", ".."))
	checks := []struct {
		name    string
		file    string
		heading string
	}{
		{
			name:    "README first-agent on-ramp",
			file:    filepath.Join(root, "README.md"),
			heading: "### First agent on-ramp",
		},
		{
			name:    "README examples list",
			file:    filepath.Join(root, "README.md"),
			heading: "## Examples",
		},
		{
			name:    "repository examples index",
			file:    filepath.Join(root, "examples", "README.md"),
			heading: "## Recommended first-agent path",
		},
		{
			name:    "repository examples wayfinding index",
			file:    filepath.Join(root, "examples", "INDEX.md"),
			heading: "## Recommended adoption path",
		},
		{
			name:    "website examples index",
			file:    filepath.Join(root, "internal", "website", "docs", "examples", "index.md"),
			heading: "## Start here",
		},
		{
			name:    "website getting-started on-ramp",
			file:    filepath.Join(root, "internal", "website", "docs", "getting-started.md"),
			heading: "### First-agent on-ramp",
		},
		{
			name:    "website quickstart next steps",
			file:    filepath.Join(root, "internal", "website", "docs", "quickstart.md"),
			heading: "## Next Steps",
		},
		{
			name:    "website docs index learn more",
			file:    filepath.Join(root, "internal", "website", "docs", "index.md"),
			heading: "## Learn More",
		},
	}

	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			section := firstMarkdownSection(t, readFile(t, check.file), check.heading)
			links := markdownLinks(section)
			if len(links) == 0 {
				t.Fatalf("%s has no Markdown links in %q", check.name, check.heading)
			}
			for _, link := range links {
				assertWayfindingTargetExists(t, root, check.file, link)
			}
		})
	}
}

func TestFirstAgentGuideChainDocumentsRequiredNextSteps(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", ".."))
	checks := []struct {
		name    string
		file    string
		markers []string
	}{
		{
			name: "no-secret transcript hands off to live build and debug",
			file: filepath.Join(root, "internal", "website", "docs", "guides", "no-secret-first-agent.md"),
			markers: []string{
				"micro agent demo",
				"go run ./examples/first-agent",
				"go run ./examples/support",
				"go test ./examples/first-agent -run TestRunFirstAgent -count=1",
				"go test ./examples/support -run TestRunSupportMockSmoke -count=1",
				"make harness",
				"micro agent preflight",
				"micro run",
				"micro chat assistant",
				"micro inspect agent assistant",
				"Debugging your agent",
				"debugging-agents.html",
			},
		},
		{
			name: "your-first-agent keeps no-secret, preflight, doctor, inspect, and debug nearby",
			file: filepath.Join(root, "internal", "website", "docs", "guides", "your-first-agent.md"),
			markers: []string{
				"no-secret-first-agent.html",
				"go run ./examples/support",
				"micro agent preflight",
				"micro agent doctor",
				"micro run",
				"micro chat assistant",
				"micro inspect agent assistant",
				"debugging-agents.html",
				"zero-to-hero.html",
			},
		},
		{
			name: "debugging guide starts at install/preflight and preserves inspect/history recovery",
			file: filepath.Join(root, "internal", "website", "docs", "guides", "debugging-agents.md"),
			markers: []string{
				"install-troubleshooting.html",
				"micro agent preflight",
				"micro agent doctor",
				"micro run",
				"micro chat",
				"micro inspect agent support",
				"micro agent history",
				"go test ./internal/harness/zero-to-hero-ci -run TestNoSecretFirstAgentDebuggingSmoke -count=1",
			},
		},
		{
			name: "zero-to-hero guide exposes the provider-free contract commands",
			file: filepath.Join(root, "internal", "website", "docs", "guides", "zero-to-hero.md"),
			markers: []string{
				"go test ./internal/harness/zero-to-hero-ci -run TestFirstAgentWayfinding -count=1",
				"micro zero-to-hero",
				"go run ./examples/first-agent",
				"go run ./examples/support",
				"make harness",
				"go test ./cmd/micro -run TestFirstAgentWalkthroughCLIBoundaries -count=1",
				"go test ./internal/harness/zero-to-hero-ci -run TestNoSecretFirstAgentDebuggingSmoke -count=1",
				"make provider-conformance-mock",
			},
		},
	}

	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			doc := readFile(t, check.file)
			for _, marker := range check.markers {
				if !strings.Contains(doc, marker) {
					t.Fatalf("%s missing required first-agent next-step marker %q", check.name, marker)
				}
				if firstAgentMarkerIsLink(marker) {
					assertWayfindingTargetExists(t, root, check.file, marker)
				}
			}
		})
	}
}

func firstAgentMarkerIsLink(marker string) bool {
	return strings.HasSuffix(marker, ".html") ||
		strings.HasSuffix(marker, ".md") ||
		strings.HasPrefix(marker, "./") ||
		strings.HasPrefix(marker, "../") ||
		strings.HasPrefix(marker, "https://github.com/micro/go-micro/")
}

func TestFirstAgentLifecycleCommandOrderIsDocumented(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", ".."))
	checks := []struct {
		name    string
		file    string
		heading string
		markers []string
	}{
		{
			name:    "0→hero guide lifecycle",
			file:    filepath.Join(root, "internal", "website", "docs", "guides", "zero-to-hero.md"),
			heading: "## What the contract covers",
			markers: []string{"micro new", "micro run", "micro chat", "micro inspect agent", "micro deploy --dry-run"},
		},
		{
			name:    "CLI docs lifecycle",
			file:    filepath.Join(root, "cmd", "micro", "cli", "cli.go"),
			heading: "const docsWayfinding",
			markers: []string{"micro agent demo", "micro run", "micro chat", "micro inspect agent", "deploy dry-run"},
		},
		{
			name:    "scaffold next steps",
			file:    filepath.Join(root, "cmd", "micro", "cli", "new", "new.go"),
			heading: "func printNextSteps",
			markers: []string{"go run .", "micro chat", "micro inspect agent", "micro agent demo", "micro docs"},
		},
	}

	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			doc := readFile(t, check.file)
			if check.heading != "" {
				start := strings.Index(doc, check.heading)
				if start == -1 {
					t.Fatalf("%s missing %q boundary", check.name, check.heading)
				}
				doc = doc[start:]
			}
			assertOrderedMarkers(t, check.name, doc, check.markers)
		})
	}
}

func TestExamplesIndexesPreserveLifecycleMap(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", ".."))
	checks := []struct {
		name    string
		file    string
		heading string
		want    []string
		ordered []string
	}{
		{
			name:    "repository examples lifecycle map",
			file:    filepath.Join(root, "examples", "README.md"),
			heading: "## Recommended first-agent path",
			want: []string{
				"hello-world",
				"0→1",
				"first-agent",
				"support",
				"services",
				"agents",
				"workflows",
				"Debugging and observability",
			},
			ordered: []string{"1. First service", "2. First agent", "3. First workflow"},
		},
		{
			name:    "website examples lifecycle map",
			file:    filepath.Join(root, "internal", "website", "docs", "examples", "index.md"),
			heading: "## Start here",
			want: []string{
				"examples/hello-world",
				"0→1",
				"examples/first-agent",
				"examples/support",
				"services",
				"agents",
				"workflows",
				"debugging-agents.html",
			},
			ordered: []string{"0→1 service", "Provider-free first agent", "0→hero lifecycle"},
		},
	}

	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			section := firstMarkdownSection(t, readFile(t, check.file), check.heading)
			for _, want := range check.want {
				if !strings.Contains(section, want) {
					t.Fatalf("%s missing lifecycle map marker %q", check.name, want)
				}
			}

			last := -1
			for _, marker := range check.ordered {
				idx := strings.Index(section, marker)
				if idx == -1 {
					t.Fatalf("%s missing ordered example marker %q", check.name, marker)
				}
				if idx < last {
					t.Fatalf("%s marker %q appeared out of order; keep examples flowing hello-world/0→1 → first-agent → support/0→hero", check.name, marker)
				}
				last = idx
			}
		})
	}
}

func TestGettingStartedDocsLeadWithNoSecretFirstRun(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", ".."))
	checks := []struct {
		name    string
		file    string
		section string
		want    []string
	}{
		{
			name:    "README quick start",
			file:    filepath.Join(root, "README.md"),
			section: "## Quick Start",
			want: []string{
				"install troubleshooting guide",
				"### Fastest start — no API key",
				"micro new helloworld",
				"micro run",
				"curl -X POST http://localhost:8080/api/helloworld/Helloworld.Call",
				"### First agent on-ramp",
				"micro agent demo",
				"### Generate from a prompt — with an LLM key",
			},
		},
		{
			name:    "CLI README",
			file:    filepath.Join(root, "cmd", "micro", "README.md"),
			section: "## Create a service",
			want: []string{
				"## Create a service",
				"micro new helloworld",
				"## Run the service",
				"micro run",
				"micro agent demo",
			},
		},
		{
			name:    "website getting started",
			file:    filepath.Join(root, "internal", "website", "docs", "getting-started.md"),
			section: "Install troubleshooting",
			want: []string{
				"Install troubleshooting",
				"## Quick Start: Scaffold, Run, Call",
				"micro new helloworld",
				"micro run",
				"curl -X POST http://localhost:8080/api/helloworld/Helloworld.Call",
				"### First-agent on-ramp",
				"micro agent demo",
				"## Generate from a Prompt — with an LLM key",
			},
		},
		{
			name:    "website quickstart",
			file:    filepath.Join(root, "internal", "website", "docs", "quickstart.md"),
			section: "## Create Your First Service",
			want: []string{
				"micro new helloworld",
				"micro run",
				"curl -X POST http://localhost:8080/api/helloworld/Helloworld.Call",
				"## Next Steps",
				"micro agent demo",
				"micro zero-to-hero",
				"guides/no-secret-first-agent.html",
				"guides/debugging-agents.html",
				"micro inspect agent <name>",
				"guides/zero-to-hero.html",
			},
		},
	}

	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			doc := readFile(t, check.file)
			if check.section != "" {
				start := strings.Index(doc, check.section)
				if start == -1 {
					t.Fatalf("%s missing %q section", check.name, check.section)
				}
				doc = doc[start:]
			}
			last := -1
			for _, want := range check.want {
				idx := strings.Index(doc, want)
				if idx == -1 {
					t.Fatalf("%s missing no-secret first-run marker %q", check.name, want)
				}
				if idx < last {
					t.Fatalf("%s marker %q appeared out of order; keep install/scaffold/run/call before provider-backed generation", check.name, want)
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
		"micro agent demo",
		"go run ./examples/first-agent",
		"go test ./examples/first-agent -run TestRunFirstAgent -count=1",
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

	debugCheckpoint := firstMarkdownSection(t, guide, "## Debug transcript checkpoint")
	for _, want := range []string{
		`micro chat assistant --prompt "Triage ticket-1 for Alice"`,
		"micro inspect agent assistant --limit 1",
		"micro agent history assistant",
		"status, event count, last event",
		"Debugging your agent",
		"debugging-agents.html",
	} {
		if !strings.Contains(debugCheckpoint, want) {
			t.Fatalf("no-secret debug transcript checkpoint missing %q", want)
		}
	}

	debuggingGuide := readFile(t, filepath.Join(root, "internal", "website", "docs", "guides", "debugging-agents.md"))
	for _, want := range []string{
		"Provider-free quickcheck",
		"go test ./internal/harness/zero-to-hero-ci -run TestNoSecretFirstAgentDebuggingSmoke -count=1",
		"micro inspect agent assistant --limit 1",
		"micro inspect agent --status done",
		"micro agent history assistant",
	} {
		if !strings.Contains(debuggingGuide, want) {
			t.Fatalf("debugging guide missing provider-free quickcheck marker %q", want)
		}
	}

	harnessReadme := readFile(t, filepath.Join(root, "internal", "harness", "zero-to-hero-ci", "README.md"))
	if !strings.Contains(harnessReadme, "go test ./internal/harness/zero-to-hero-ci -run TestNoSecretFirstAgentDebuggingSmoke -count=1") {
		t.Fatal("0→hero harness README does not expose the agent debugging quickcheck command")
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

func TestNoSecretFirstAgentDebuggingSmoke(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", ".."))
	home := t.TempDir()
	storeDir := filepath.Join(home, "micro", "store")
	st := store.NewFileStore(store.DirOption(storeDir))

	seedNoSecretAgentDebuggingState(t, st)
	if err := st.Close(); err != nil {
		t.Fatalf("close seeded store: %v", err)
	}

	micro := buildMicroBinary(t, root)

	for _, tc := range []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "demo advertises provider-free debug path",
			args: []string{"agent", "demo"},
			want: []string{"No-secret first-agent demo", "provider-free", "run history", "micro inspect agent <name>"},
		},
		{
			name: "inspect shows seeded run history",
			args: []string{"inspect", "agent", "assistant", "--limit", "1"},
			want: []string{`Agent "assistant" runs`, "run-debug-smoke", "status=done", "events=3", "last=done", "trace=trace-debug-"},
		},
		{
			name: "inspect filters documented statuses",
			args: []string{"inspect", "agent", "--status", "done", "--json", "assistant"},
			want: []string{"run-debug-smoke", `"status": "done"`, `"trace_id": "trace-debug-smoke"`},
		},
		{
			name: "agent history shows memory and run index",
			args: []string{"agent", "history", "assistant"},
			want: []string{"user:", "Triage ticket-1", "assistant:", "ticket-1 is ready", "Runs:", "run-debug-smoke", "status=done"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out := runMicroCLIWithHome(t, micro, home, tc.args...)
			for _, want := range tc.want {
				if !strings.Contains(out, want) {
					t.Fatalf("micro %s output missing %q:\n%s", strings.Join(tc.args, " "), want, out)
				}
			}
		})
	}
}

func TestFirstAgentCLIChatInspectFixture(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", ".."))
	absRoot, err := filepath.Abs(root)
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}

	workspace := t.TempDir()
	home := filepath.Join(workspace, "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatalf("create fixture home: %v", err)
	}
	writeFile(t, filepath.Join(workspace, "go.mod"), "module example.com/first-agent-cli-fixture\n\ngo 1.24\n\nrequire go-micro.dev/v6 v6.0.0\n\nreplace go-micro.dev/v6 => "+filepath.ToSlash(absRoot)+"\n")
	writeFile(t, filepath.Join(workspace, "main.go"), firstAgentCLIFixtureSource())
	runInWorkspace(t, workspace, "go", "mod", "tidy")

	micro := buildMicroBinary(t, absRoot)
	fixtureBin := filepath.Join(workspace, "first-agent-cli-fixture")
	runInWorkspace(t, workspace, "go", "build", "-o", fixtureBin, ".")
	fixture := exec.Command(fixtureBin)
	fixture.Dir = workspace
	fixture.Env = append(os.Environ(),
		"HOME="+home,
		"MICRO_AI_API_KEY=",
		"OPENAI_API_KEY=",
		"ANTHROPIC_API_KEY=",
		"GEMINI_API_KEY=",
	)
	var fixtureOut lockedBuffer
	fixture.Stdout = &fixtureOut
	fixture.Stderr = &fixtureOut
	if err := fixture.Start(); err != nil {
		t.Fatalf("start first-agent CLI fixture: %v\n%s", err, fixtureOut.String())
	}
	defer func() {
		if fixture.Process != nil {
			_ = fixture.Process.Signal(os.Interrupt)
			_ = fixture.Process.Kill()
		}
	}()

	waitForCLIOutput(t, &fixtureOut, "first-agent fixture ready", 15*time.Second)
	waitForRegisteredAgent(t, micro, home, "assistant", &fixtureOut, 15*time.Second)

	chat := runMicroCLIWithHome(t, micro, home, "chat", "--prompt", "Summarize my first-agent next steps", "assistant")
	for _, want := range []string{"assistant:", "install the CLI", "run a service", "chat with an agent"} {
		if !strings.Contains(chat, want) {
			t.Fatalf("micro chat assistant output missing %q:\n%s\nfixture output:\n%s", want, chat, fixtureOut.String())
		}
	}

	stopFixture(t, fixture)

	inspect := runMicroCLIWithHome(t, micro, home, "inspect", "agent", "assistant", "--limit", "1")
	for _, want := range []string{`Agent "assistant" runs`, "status=done", "last=done"} {
		if !strings.Contains(inspect, want) {
			t.Fatalf("micro inspect agent assistant output missing %q:\n%s\nfixture output:\n%s", want, inspect, fixtureOut.String())
		}
	}
}

func stopFixture(t *testing.T, cmd *exec.Cmd) {
	t.Helper()
	if cmd.Process == nil {
		return
	}
	done := make(chan error, 1)
	_ = cmd.Process.Signal(os.Interrupt)
	go func() { done <- cmd.Wait() }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		_ = cmd.Process.Kill()
		<-done
	}
}

func waitForCLIOutput(t *testing.T, out *lockedBuffer, want string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if strings.Contains(out.String(), want) {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for fixture output %q; got:\n%s", want, out.String())
}

func waitForRegisteredAgent(t *testing.T, micro, home, agent string, fixtureOut *lockedBuffer, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var lastOut []byte
	var lastErr error
	for time.Now().Before(deadline) {
		cmd := exec.Command(micro, "agent", "list")
		cmd.Env = microCLIEnv(home)
		lastOut, lastErr = cmd.CombinedOutput()
		if lastErr == nil && strings.Contains(string(lastOut), agent) {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for registered agent %q; last micro agent list error: %v\nlast output:\n%s\nfixture output:\n%s", agent, lastErr, lastOut, fixtureOut.String())
}

type lockedBuffer struct {
	mu sync.Mutex
	b  strings.Builder
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.String()
}

func firstAgentCLIFixtureSource() string {
	return `package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"go-micro.dev/v6/agent"
	"go-micro.dev/v6/ai"
	"go-micro.dev/v6/service"
	"go-micro.dev/v6/store"
)

type NotesService struct{}
type ListNotesRequest struct{}
type ListNotesResponse struct { Notes []string ` + "`json:\"notes\" description:\"Notes the assistant can summarize\"`" + ` }

func (s *NotesService) List(ctx context.Context, req *ListNotesRequest, rsp *ListNotesResponse) error {
	rsp.Notes = []string{"Install the CLI", "Run a service", "Chat with an agent"}
	return nil
}

type mockModel struct{ opts ai.Options }
func newMock(opts ...ai.Option) ai.Model { m := &mockModel{}; _ = m.Init(opts...); return m }
func (m *mockModel) Init(opts ...ai.Option) error { for _, o := range opts { o(&m.opts) }; return nil }
func (m *mockModel) Options() ai.Options { return m.opts }
func (m *mockModel) String() string { return "first-agent-cli-fixture" }
func (m *mockModel) Stream(context.Context, *ai.Request, ...ai.GenerateOption) (ai.Stream, error) { return nil, fmt.Errorf("stream unsupported") }
func (m *mockModel) Generate(ctx context.Context, req *ai.Request, _ ...ai.GenerateOption) (*ai.Response, error) {
	for _, tool := range req.Tools { if strings.Contains(tool.Name, "List") && m.opts.ToolHandler != nil { m.opts.ToolHandler(ctx, ai.ToolCall{ID:"list-notes", Name: tool.Name, Input: map[string]any{}}); break } }
	return &ai.Response{Answer: "assistant: your first agent should install the CLI, run a service, then chat with an agent."}, nil
}

func main() {
	ai.Register("first-agent-cli-fixture", newMock)
	home, _ := os.UserHomeDir()
	st := store.NewFileStore(store.DirOption(filepath.Join(home, "micro", "store")))
	defer st.Close()

	svc := service.New(service.Name("notes"), service.Address("127.0.0.1:0"))
	if err := svc.Handle(&NotesService{}); err != nil { panic(err) }
	go func() { if err := svc.Run(); err != nil { fmt.Println(err); os.Exit(1) } }()
	defer svc.Server().Stop()

	a := agent.New(agent.Name("assistant"), agent.Address("127.0.0.1:0"), agent.Services("notes"), agent.Provider("first-agent-cli-fixture"), agent.WithStore(st))
	go func() { if err := a.Run(); err != nil { fmt.Println(err); os.Exit(1) } }()
	defer a.Stop()

	fmt.Println("first-agent fixture ready")
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
}
`
}

func seedNoSecretAgentDebuggingState(t *testing.T, st store.Store) {
	t.Helper()
	scoped := store.Scope(st, "agent", "assistant")
	runID := "run-debug-smoke"
	events := []goagent.RunEvent{
		{Time: time.Unix(1700000000, 0), RunID: runID, Agent: "assistant", TraceID: "trace-debug-smoke", Kind: "run", Name: "ask"},
		{Time: time.Unix(1700000001, 0), RunID: runID, Agent: "assistant", TraceID: "trace-debug-smoke", Kind: "model", Provider: "mock", Model: "first-agent-mock"},
		{Time: time.Unix(1700000002, 0), RunID: runID, Agent: "assistant", TraceID: "trace-debug-smoke", Kind: "done", Name: "answer"},
	}
	for _, event := range events {
		b, err := json.Marshal(event)
		if err != nil {
			t.Fatal(err)
		}
		key := "runs/" + event.RunID + "/" + event.Time.Format("20060102150405.000000000") + "-" + event.Kind
		if err := scoped.Write(&store.Record{Key: key, Value: b}); err != nil {
			t.Fatalf("seed run event: %v", err)
		}
	}

	mem := goagent.NewMemory(scoped, "history", 10)
	mem.Add("user", "Triage ticket-1 for Alice")
	mem.Add("assistant", "ticket-1 is ready for Alice without provider secrets")
}

func buildMicroBinary(t *testing.T, root string) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "micro")
	cmd := exec.Command("go", "build", "-o", bin, "./cmd/micro")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build micro CLI failed: %v\n%s", err, out)
	}
	return bin
}

func runMicroCLIWithHome(t *testing.T, micro, home string, args ...string) string {
	t.Helper()
	cmd := exec.Command(micro, args...)
	cmd.Env = microCLIEnv(home)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("micro %s failed: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

func microCLIEnv(home string) []string {
	return append(os.Environ(),
		"HOME="+home,
		"MICRO_AI_API_KEY=",
		"OPENAI_API_KEY=",
		"ANTHROPIC_API_KEY=",
		"GEMINI_API_KEY=",
	)
}

func TestFirstAgentWayfindingTargetsExist(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", ".."))
	for _, target := range []string{
		"examples/README.md",
		"examples/first-agent/README.md",
		"internal/website/docs/examples/index.md",
		"internal/website/docs/guides/no-secret-first-agent.md",
		"internal/website/docs/guides/your-first-agent.md",
		"internal/website/docs/guides/debugging-agents.md",
		"internal/website/docs/guides/zero-to-hero.md",
	} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(target))); err != nil {
			t.Fatalf("first-agent wayfinding target %s disappeared: %v", target, err)
		}
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

var markdownLinkRE = regexp.MustCompile(`\[[^\]]+\]\(([^)#?]+)(?:[#?][^)]*)?\)`)

func markdownLinks(section string) []string {
	matches := markdownLinkRE.FindAllStringSubmatch(section, -1)
	links := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) > 1 {
			links = append(links, match[1])
		}
	}
	return links
}

func containsWayfindingMarker(section, marker string) bool {
	if strings.Contains(section, marker) {
		return true
	}
	switch marker {
	case "examples/INDEX.md":
		return strings.Contains(section, "examples/INDEX.md") ||
			strings.Contains(section, "Recommended adoption path") ||
			strings.Contains(section, "examples wayfinding index") ||
			strings.Contains(section, "Examples wayfinding index") ||
			strings.Contains(section, "./INDEX.md")
	case "examples/first-agent":
		return strings.Contains(section, "examples/first-agent") ||
			strings.Contains(section, "./first-agent")
	case "examples/support":
		return strings.Contains(section, "examples/support") ||
			strings.Contains(section, "./support")
	case "no-secret-first-agent":
		return strings.Contains(section, "no-secret-first-agent") ||
			strings.Contains(section, "No-secret First Agent")
	case "your-first-agent":
		return strings.Contains(section, "your-first-agent") ||
			strings.Contains(section, "Your First Agent")
	case "debugging-agents":
		return strings.Contains(section, "debugging-agents") ||
			strings.Contains(section, "Debugging your agent")
	case "zero-to-hero":
		return strings.Contains(section, "zero-to-hero") ||
			strings.Contains(section, "0→hero")
	default:
		if strings.HasPrefix(marker, "micro inspect agent ") {
			return strings.Contains(section, "micro inspect agent ")
		}
		return false
	}
}

func assertWayfindingTargetExists(t *testing.T, root, sourceFile, link string) {
	t.Helper()
	if !strings.Contains(link, "/") && !strings.Contains(link, ".") {
		return
	}
	if strings.HasPrefix(link, "http://") || strings.HasPrefix(link, "https://") {
		switch {
		case strings.HasPrefix(link, "https://go-micro.dev/docs/"):
			link = strings.TrimPrefix(link, "https://go-micro.dev/docs/")
			link = filepath.ToSlash(filepath.Join("internal", "website", "docs", strings.TrimSuffix(link, ".html")+".md"))
		case strings.HasPrefix(link, "https://github.com/micro/go-micro/tree/master/"):
			link = strings.TrimPrefix(link, "https://github.com/micro/go-micro/tree/master/")
		default:
			return
		}
	} else if strings.HasSuffix(link, ".html") {
		sourceDir := filepath.Dir(sourceFile)
		websiteDocs := filepath.Join(root, "internal", "website", "docs")
		resolved := filepath.Clean(filepath.Join(sourceDir, filepath.FromSlash(link)))
		if rel, err := filepath.Rel(websiteDocs, resolved); err == nil && !strings.HasPrefix(rel, "..") {
			link = filepath.ToSlash(filepath.Join("internal", "website", "docs", strings.TrimSuffix(rel, ".html")+".md"))
		}
	} else if strings.HasPrefix(link, ".") {
		target := filepath.Clean(filepath.Join(filepath.Dir(sourceFile), filepath.FromSlash(link)))
		if _, err := os.Stat(target); err != nil {
			t.Fatalf("first-agent wayfinding link %q in %s resolves to missing target %s: %v", link, sourceFile, target, err)
		}
		return
	}

	target := filepath.Join(root, filepath.FromSlash(link))
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("first-agent wayfinding link %q in %s resolves to missing target %s: %v", link, sourceFile, target, err)
	}
}

func assertOrderedMarkers(t *testing.T, name, doc string, markers []string) {
	t.Helper()
	last := -1
	for _, marker := range markers {
		idx := strings.Index(doc, marker)
		if idx == -1 {
			t.Fatalf("%s missing lifecycle command marker %q", name, marker)
		}
		if idx < last {
			t.Fatalf("%s marker %q appeared out of order; keep scaffold → run → chat → inspect → deploy discoverable", name, marker)
		}
		last = idx
	}
}
