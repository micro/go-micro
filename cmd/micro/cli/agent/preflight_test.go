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
	for _, want := range []string{"✗ Go toolchain", "Install Go 1.24", "✗ micro binary", "✗ provider API key", "ANTHROPIC_API_KEY", "✗ local port :8080", "micro run --address"} {
		if !strings.Contains(got, want) {
			t.Fatalf("output missing %q:\n%s", want, got)
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
