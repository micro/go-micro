package zerotoheroci

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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
		"go test ./examples/first-agent -run TestRunFirstAgent -count=1",
		"go test ./examples/support -run 'TestRunSupportMockSmoke|TestZeroToHeroReadmeDocumentsLifecycle' -count=1",
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
				"micro zero-to-hero",
				"internal/website/docs/guides/no-secret-first-agent.md",
				"internal/website/docs/guides/your-first-agent.md",
				"internal/website/docs/guides/debugging-agents.md",
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
				"micro zero-to-hero",
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
				assertWayfindingTargetExists(t, root, check.file, link)
				if idx < last {
					t.Fatalf("%s link %q appeared out of order; expected no-secret → first-agent → debugging → 0→hero", check.name, link)
				}
				last = idx
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
			name:    "website examples index",
			file:    filepath.Join(root, "internal", "website", "docs", "examples", "index.md"),
			heading: "## Start here",
		},
		{
			name:    "website getting-started on-ramp",
			file:    filepath.Join(root, "internal", "website", "docs", "getting-started.md"),
			heading: "### First-agent on-ramp",
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

	readme := readFile(t, filepath.Join(root, "README.md"))
	if !strings.Contains(readme, "internal/website/docs/guides/no-secret-first-agent.md") {
		t.Fatal("README does not point to the no-secret first-agent transcript")
	}

	firstAgent := readFile(t, filepath.Join(root, "internal", "website", "docs", "guides", "your-first-agent.md"))
	if !strings.Contains(firstAgent, "no-secret-first-agent.html") {
		t.Fatal("Your First Agent guide does not point to the no-secret transcript")
	}
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
