package agent

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	goagent "go-micro.dev/v6/agent"
	"go-micro.dev/v6/registry"
)

func doctorHTTP(status int, body string) func(string) (*http.Response, error) {
	return func(string) (*http.Response, error) {
		return &http.Response{StatusCode: status, Status: "200 OK", Body: io.NopCloser(strings.NewReader(body))}, nil
	}
}

func TestRunAgentDoctorPassesWhenRecoveryBoundariesReachable(t *testing.T) {
	deps := doctorDeps{
		getenv: func(key string) string {
			if key == "MICRO_AI_API_KEY" {
				return "set"
			}
			return ""
		},
		httpGet: doctorHTTP(200, `{"provider":"anthropic","model":"claude"}`),
		listServices: func() ([]*registry.Service, error) {
			return []*registry.Service{{Name: "assistant"}}, nil
		},
		getService: func(name string) ([]*registry.Service, error) {
			return []*registry.Service{{Name: name, Metadata: map[string]string{"type": "agent"}}}, nil
		},
		listRuns: func(name string) ([]goagent.RunSummary, error) {
			return []goagent.RunSummary{{RunID: "run-1", Status: "done"}}, nil
		},
	}
	var out bytes.Buffer
	if err := runAgentDoctor(&out, deps, "http://example.test"); err != nil {
		t.Fatalf("runAgentDoctor() error = %v\n%s", err, out.String())
	}
	got := out.String()
	for _, want := range []string{"First-agent recovery doctor", "✓ gateway /agent", "✓ chat settings endpoint", "✓ agent registration", "✓ inspect run history", "✓ provider configuration", "Ready:"} {
		if !strings.Contains(got, want) {
			t.Fatalf("output missing %q:\n%s", want, got)
		}
	}
}

func TestRunAgentDoctorReportsActionableRecoveryFailures(t *testing.T) {
	deps := doctorDeps{
		getenv:  func(string) string { return "" },
		httpGet: func(string) (*http.Response, error) { return nil, errors.New("connection refused") },
		listServices: func() ([]*registry.Service, error) {
			return []*registry.Service{{Name: "greeter"}}, nil
		},
		getService: func(name string) ([]*registry.Service, error) {
			return []*registry.Service{{Name: name}}, nil
		},
		listRuns: func(name string) ([]goagent.RunSummary, error) { return nil, nil },
	}
	var out bytes.Buffer
	err := runAgentDoctor(&out, deps, "http://localhost:8080")
	if err == nil {
		t.Fatal("runAgentDoctor() error = nil")
	}
	got := out.String()
	for _, want := range []string{"✗ gateway /agent", "micro run", "✗ chat settings endpoint", "✗ agent registration", "micro agent list", "✗ inspect run history", "micro inspect agent <name>", "✗ provider configuration", "docs/guides/no-secret-first-agent.html"} {
		if !strings.Contains(got, want) {
			t.Fatalf("output missing %q:\n%s", want, got)
		}
	}
}

func TestAgentQuickcheckPrintsProviderFreeFailureModeBreadcrumbs(t *testing.T) {
	got := firstAgentQuickChecksHelp
	for _, want := range []string{
		"First-agent failure-mode quick checks",
		"scaffold -> run -> chat -> inspect",
		"micro agent preflight",
		"micro run",
		"micro agent doctor",
		"micro inspect agent <name>",
		"micro runs <name>",
		"micro agent demo",
		"go test ./internal/harness/zero-to-hero-ci -run TestNoSecretFirstAgentTranscript -count=1",
		"go test ./internal/harness/zero-to-hero-ci -run TestNoSecretFirstAgentDebuggingSmoke -count=1",
		"debugging-agents.html",
		"no-secret-first-agent.html",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("quickcheck output missing %q:\n%s", want, got)
		}
	}
}
