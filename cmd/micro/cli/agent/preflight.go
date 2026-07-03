package agent

import (
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strings"

	"go-micro.dev/v6/cmd"
)

type preflightCheck struct {
	Name   string
	OK     bool
	Detail string
	Fix    string
	Next   string
}

type preflightDeps struct {
	lookPath      func(string) (string, error)
	commandOutput func(string, ...string) ([]byte, error)
	executable    func() (string, error)
	version       func() string
	getenv        func(string) string
	listen        func(string, string) (net.Listener, error)
}

func defaultPreflightDeps() preflightDeps {
	return preflightDeps{
		lookPath:      exec.LookPath,
		commandOutput: func(name string, args ...string) ([]byte, error) { return exec.Command(name, args...).CombinedOutput() },
		executable:    os.Executable,
		version:       func() string { return cmd.App().Version },
		getenv:        os.Getenv,
		listen:        net.Listen,
	}
}

func runAgentPreflight(w io.Writer, deps preflightDeps) error {
	checks := agentPreflightChecks(deps)
	failures := 0
	fmt.Fprintln(w, "First-agent preflight")
	for _, check := range checks {
		mark := "✓"
		if !check.OK {
			mark = "✗"
			failures++
		}
		fmt.Fprintf(w, "  %s %s — %s\n", mark, check.Name, check.Detail)
		if !check.OK && check.Fix != "" {
			fmt.Fprintf(w, "    Fix: %s\n", check.Fix)
		}
		if !check.OK && check.Next != "" {
			fmt.Fprintf(w, "    Next: %s\n", check.Next)
		}
	}
	if failures > 0 {
		return fmt.Errorf("first-agent preflight failed: %d check(s) need attention", failures)
	}
	fmt.Fprintln(w, "\nReady for the first-agent walkthrough: micro run, then open http://localhost:8080/agent or use micro chat.")
	return nil
}

func agentPreflightChecks(deps preflightDeps) []preflightCheck {
	if deps.lookPath == nil {
		deps.lookPath = exec.LookPath
	}
	if deps.commandOutput == nil {
		deps.commandOutput = func(name string, args ...string) ([]byte, error) { return exec.Command(name, args...).CombinedOutput() }
	}
	if deps.executable == nil {
		deps.executable = os.Executable
	}
	if deps.version == nil {
		deps.version = func() string { return cmd.App().Version }
	}
	if deps.getenv == nil {
		deps.getenv = os.Getenv
	}
	if deps.listen == nil {
		deps.listen = net.Listen
	}

	checks := []preflightCheck{checkGoToolchain(deps), checkMicroBinary(deps), checkProviderKey(deps), checkPortAvailable(deps, ":8080", "micro run gateway and /agent playground")}
	return checks
}

func checkGoToolchain(deps preflightDeps) preflightCheck {
	path, err := deps.lookPath("go")
	if err != nil {
		return preflightCheck{Name: "Go toolchain", Detail: "go was not found on PATH", Fix: "Install Go 1.24 or newer from https://go.dev/doc/install and ensure go is on PATH.", Next: "After installing Go, rerun micro agent preflight, then continue with docs/guides/your-first-agent.html."}
	}
	out, err := deps.commandOutput("go", "version")
	if err != nil {
		return preflightCheck{Name: "Go toolchain", Detail: strings.TrimSpace(string(out)), Fix: "Ensure the go command runs successfully (try `go version`) before starting the agent walkthrough.", Next: "Use docs/guides/debugging-agents.html after the toolchain check passes if an agent run still fails."}
	}
	version := firstLine(out)
	if !goVersionAtLeast(version, 1, 24) {
		return preflightCheck{Name: "Go toolchain", Detail: fmt.Sprintf("%s (%s)", version, path), Fix: "Upgrade to Go 1.24 or newer before running generated services.", Next: "Rerun micro agent preflight, then continue with docs/guides/your-first-agent.html."}
	}
	return preflightCheck{Name: "Go toolchain", OK: true, Detail: fmt.Sprintf("%s (%s)", version, path)}
}

func checkMicroBinary(deps preflightDeps) preflightCheck {
	exe, err := deps.executable()
	if err != nil || exe == "" {
		return preflightCheck{Name: "micro binary", Detail: "micro executable path is unavailable", Fix: "Install the micro CLI or run this check through `go run ./cmd/micro agent preflight` from the repository.", Next: "Then follow docs/getting-started.html for the scaffold -> run path."}
	}
	version := deps.version()
	if version == "" {
		version = "version unavailable"
	}
	return preflightCheck{Name: "micro binary", OK: true, Detail: fmt.Sprintf("%s (%s)", version, exe)}
}

func checkProviderKey(deps preflightDeps) preflightCheck {
	keys := []string{"MICRO_AI_API_KEY", "ANTHROPIC_API_KEY", "OPENAI_API_KEY", "GEMINI_API_KEY", "GROQ_API_KEY", "MISTRAL_API_KEY", "TOGETHER_API_KEY", "ATLASCLOUD_API_KEY"}
	var found []string
	for _, k := range keys {
		if deps.getenv(k) != "" {
			found = append(found, k)
		}
	}
	if len(found) == 0 {
		return preflightCheck{Name: "provider API key", Detail: "no supported provider key found", Fix: "Export MICRO_AI_API_KEY or a provider key such as ANTHROPIC_API_KEY before running provider-backed agents.", Next: "For a no-secret path, run the mock-model walkthrough in docs/guides/no-secret-first-agent.html; for real providers, see docs/guides/debugging-agents.html#provider-failures."}
	}
	return preflightCheck{Name: "provider API key", OK: true, Detail: "found " + strings.Join(found, ", ")}
}

func checkPortAvailable(deps preflightDeps, addr, use string) preflightCheck {
	ln, err := deps.listen("tcp", addr)
	if err != nil {
		return preflightCheck{Name: "local port " + addr, Detail: "busy or unavailable for " + use, Fix: "Stop the process using " + addr + " (for example, `lsof -i :8080`) or run `micro run --address` with a free port.", Next: "Once the gateway starts, open http://localhost:8080/agent or continue with docs/guides/your-first-agent.html#chat-with-your-agent."}
	}
	_ = ln.Close()
	return preflightCheck{Name: "local port " + addr, OK: true, Detail: "available for " + use}
}

func firstLine(b []byte) string {
	s := strings.TrimSpace(string(b))
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

func goVersionAtLeast(line string, wantMajor, wantMinor int) bool {
	idx := strings.Index(line, "go1.")
	if idx < 0 {
		return false
	}
	var major, minor int
	if _, err := fmt.Sscanf(line[idx:], "go%d.%d", &major, &minor); err != nil {
		return false
	}
	if major != wantMajor {
		return major > wantMajor
	}
	return minor >= wantMinor
}
