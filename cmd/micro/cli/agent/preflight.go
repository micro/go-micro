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
		return preflightCheck{Name: "Go toolchain", Fix: "Install Go 1.24 or newer and ensure go is on PATH."}
	}
	out, err := deps.commandOutput("go", "version")
	if err != nil {
		return preflightCheck{Name: "Go toolchain", Detail: strings.TrimSpace(string(out)), Fix: "Ensure the go command runs successfully."}
	}
	return preflightCheck{Name: "Go toolchain", OK: true, Detail: fmt.Sprintf("%s (%s)", firstLine(out), path)}
}

func checkMicroBinary(deps preflightDeps) preflightCheck {
	exe, err := deps.executable()
	if err != nil || exe == "" {
		return preflightCheck{Name: "micro binary", Fix: "Install the micro CLI or run this check through go run ./cmd/micro agent preflight."}
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
		return preflightCheck{Name: "provider API key", Detail: "no supported provider key found", Fix: "Export MICRO_AI_API_KEY or a provider key such as ANTHROPIC_API_KEY before running provider-backed agents."}
	}
	return preflightCheck{Name: "provider API key", OK: true, Detail: "found " + strings.Join(found, ", ")}
}

func checkPortAvailable(deps preflightDeps, addr, use string) preflightCheck {
	ln, err := deps.listen("tcp", addr)
	if err != nil {
		return preflightCheck{Name: "local port " + addr, Detail: "busy or unavailable for " + use, Fix: "Stop the process using " + addr + " or run micro run --address with a free port."}
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
