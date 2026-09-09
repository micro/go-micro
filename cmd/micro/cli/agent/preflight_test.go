package agent

import (
	"bytes"
	"errors"
	"net"
	"strings"
	"testing"
)

type stubListener struct{}

func (stubListener) Accept() (net.Conn, error) { return nil, errors.New("closed") }
func (stubListener) Close() error              { return nil }
func (stubListener) Addr() net.Addr            { return stubAddr(":8080") }

type stubAddr string

func (a stubAddr) Network() string { return "tcp" }
func (a stubAddr) String() string  { return string(a) }

func TestRunAgentPreflightPassesWithKeyAndFreePort(t *testing.T) {
	deps := preflightDeps{
		lookPath: func(name string) (string, error) { return "/usr/bin/" + name, nil },
		commandOutput: func(name string, args ...string) ([]byte, error) {
			return []byte("go version go1.24.0 linux/amd64\n"), nil
		},
		executable: func() (string, error) { return "/usr/local/bin/micro", nil },
		getenv: func(key string) string {
			if key == "ANTHROPIC_API_KEY" {
				return "set"
			}
			return ""
		},
		listen: func(network, address string) (net.Listener, error) { return stubListener{}, nil },
	}

	var out bytes.Buffer
	if err := runAgentPreflight(&out, deps); err != nil {
		t.Fatalf("runAgentPreflight() error = %v", err)
	}
	got := out.String()
	for _, want := range []string{"First-agent preflight", "✓ Go toolchain", "✓ micro binary", "✓ provider API key", "✓ local port :8080", "Ready for the first-agent walkthrough"} {
		if !strings.Contains(got, want) {
			t.Fatalf("output missing %q:\n%s", want, got)
		}
	}
}

func TestRunAgentPreflightReportsActionableFailures(t *testing.T) {
	deps := preflightDeps{
		lookPath:   func(name string) (string, error) { return "", errors.New("not found") },
		executable: func() (string, error) { return "", errors.New("unknown") },
		getenv:     func(key string) string { return "" },
		listen:     func(network, address string) (net.Listener, error) { return nil, errors.New("in use") },
	}

	var out bytes.Buffer
	err := runAgentPreflight(&out, deps)
	if err == nil {
		t.Fatal("runAgentPreflight() error = nil")
	}
	got := out.String()
	for _, want := range []string{"✗ Go toolchain", "go was not found on PATH", "https://go.dev/doc/install", "docs/guides/your-first-agent.html", "✗ micro binary", "go run ./cmd/micro agent preflight", "✗ provider API key", "docs/guides/no-secret-first-agent.html", "docs/guides/debugging-agents.html#provider-failures", "✗ local port :8080", "lsof -i :8080", "micro run --address"} {
		if !strings.Contains(got, want) {
			t.Fatalf("output missing %q:\n%s", want, got)
		}
	}
}

func TestRunAgentPreflightReportsOldGoVersion(t *testing.T) {
	deps := preflightDeps{
		lookPath: func(name string) (string, error) { return "/usr/bin/" + name, nil },
		commandOutput: func(name string, args ...string) ([]byte, error) {
			return []byte("go version go1.23.9 linux/amd64\n"), nil
		},
		executable: func() (string, error) { return "/usr/local/bin/micro", nil },
		getenv: func(key string) string {
			if key == "ANTHROPIC_API_KEY" {
				return "set"
			}
			return ""
		},
		listen: func(network, address string) (net.Listener, error) { return stubListener{}, nil },
	}

	var out bytes.Buffer
	err := runAgentPreflight(&out, deps)
	if err == nil {
		t.Fatal("runAgentPreflight() error = nil")
	}
	got := out.String()
	for _, want := range []string{"✗ Go toolchain", "go1.23.9", "Upgrade to Go 1.24 or newer", "Rerun micro agent preflight"} {
		if !strings.Contains(got, want) {
			t.Fatalf("output missing %q:\n%s", want, got)
		}
	}
}

func TestGoVersionAtLeast(t *testing.T) {
	tests := []struct {
		line string
		want bool
	}{
		{line: "go version go1.24.0 linux/amd64", want: true},
		{line: "go version go1.25.1 linux/amd64", want: true},
		{line: "go version go1.23.9 linux/amd64", want: false},
		{line: "unexpected", want: false},
	}
	for _, tt := range tests {
		if got := goVersionAtLeast(tt.line, 1, 24); got != tt.want {
			t.Fatalf("goVersionAtLeast(%q) = %v, want %v", tt.line, got, tt.want)
		}
	}
}

func TestFirstLine(t *testing.T) {
	if got := firstLine([]byte("one\ntwo")); got != "one" {
		t.Fatalf("firstLine() = %q", got)
	}
	if got := firstLine([]byte(" single ")); got != "single" {
		t.Fatalf("firstLine() = %q", got)
	}
}
