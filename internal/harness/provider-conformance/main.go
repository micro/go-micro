// Provider conformance runs the same end-to-end harnesses across model
// providers whose API keys are configured. Missing keys are skipped so the
// command is safe in local development and scheduled CI; a configured provider
// that fails any harness makes the command fail.
//
// Run all live providers with configured keys:
//
//	go run ./internal/harness/provider-conformance
//
// Run the deterministic mock path only:
//
//	go run ./internal/harness/provider-conformance -providers mock
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

var providerEnv = map[string]string{
	"anthropic":  "ANTHROPIC_API_KEY",
	"openai":     "OPENAI_API_KEY",
	"gemini":     "GEMINI_API_KEY",
	"groq":       "GROQ_API_KEY",
	"mistral":    "MISTRAL_API_KEY",
	"together":   "TOGETHER_API_KEY",
	"atlascloud": "ATLASCLOUD_API_KEY",
}

func main() {
	providersFlag := flag.String("providers", "anthropic,openai,gemini,groq,mistral,together,atlascloud", "comma-separated providers to check; use mock for deterministic local checks")
	harnessesFlag := flag.String("harnesses", "universe,agent-flow,plan-delegate", "comma-separated harness names under internal/harness")
	timeoutFlag := flag.Duration("timeout", 10*time.Minute, "timeout per provider/harness run")
	requireConfiguredFlag := flag.Bool("require-configured", false, "fail when a selected live provider is missing an API key")
	flag.Parse()

	providers := splitCSV(*providersFlag)
	harnesses := splitCSV(*harnessesFlag)
	if err := validateSelection(providers, harnesses); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	var ran, skipped, failed int
	for _, provider := range providers {
		if provider != "mock" && providerKey(provider) == "" {
			msg := fmt.Sprintf("set MICRO_AI_API_KEY or %s", providerEnv[provider])
			if *requireConfiguredFlag {
				fmt.Printf("FAIL %s: missing API key (%s)\n", provider, msg)
				failed++
			} else {
				fmt.Printf("- %s: skipped (%s)\n", provider, msg)
				skipped++
			}
			continue
		}

		for _, harness := range harnesses {
			fmt.Printf("\n==> %s / %s\n", provider, harness)
			if err := runHarness(provider, harness, *timeoutFlag); err != nil {
				fmt.Printf("FAIL %s / %s: %v\n", provider, harness, err)
				failed++
				continue
			}
			ran++
		}
	}

	fmt.Printf("\nprovider conformance: %d passed, %d skipped providers, %d failed\n", ran, skipped, failed)
	if failed > 0 {
		os.Exit(1)
	}
}

func validateSelection(providers, harnesses []string) error {
	if len(providers) == 0 {
		return fmt.Errorf("no providers selected")
	}
	if len(harnesses) == 0 {
		return fmt.Errorf("no harnesses selected")
	}

	for _, provider := range providers {
		if provider == "mock" {
			continue
		}
		if _, ok := providerEnv[provider]; !ok {
			return fmt.Errorf("unknown provider %q (known: %s)", provider, knownProviders())
		}
	}

	for _, harness := range harnesses {
		if strings.Contains(harness, string(os.PathSeparator)) || harness == "." || harness == ".." {
			return fmt.Errorf("invalid harness name %q", harness)
		}
		info, err := os.Stat(filepath.Join(repoRoot(), "internal", "harness", harness))
		if err != nil {
			return fmt.Errorf("unknown harness %q: %w", harness, err)
		}
		if !info.IsDir() {
			return fmt.Errorf("harness %q is not a directory", harness)
		}
	}

	return nil
}

func repoRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	for {
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			return wd
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			return "."
		}
		wd = parent
	}
}

func knownProviders() string {
	providers := make([]string, 0, len(providerEnv)+1)
	providers = append(providers, "mock")
	for provider := range providerEnv {
		providers = append(providers, provider)
	}
	slices.Sort(providers)
	return strings.Join(providers, ",")
}

func splitCSV(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func providerKey(provider string) string {
	if v := os.Getenv("MICRO_AI_API_KEY"); v != "" {
		return v
	}
	return os.Getenv(providerEnv[provider])
}

func localRPCEnv(env []string) []string {
	filtered := env[:0]
	for _, kv := range env {
		key, _, ok := strings.Cut(kv, "=")
		if !ok {
			filtered = append(filtered, kv)
			continue
		}
		switch strings.ToUpper(key) {
		case "HTTP_PROXY", "HTTPS_PROXY", "NO_PROXY":
			continue
		default:
			filtered = append(filtered, kv)
		}
	}
	return append(filtered, "HTTP_PROXY=", "HTTPS_PROXY=", "NO_PROXY=*")
}

func runHarness(provider, harness string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Build the harness to a temp binary and run that, rather than `go run`:
	// `go run` launches the compiled binary as a child it does not kill on
	// context cancellation, so a harness that starts local services could
	// outlive the timeout. Running the binary ourselves keeps the timeout
	// honest — canceling the context kills the process that does the work.
	binDir, err := os.MkdirTemp("", "harness-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(binDir)
	binPath := filepath.Join(binDir, harness)

	build := exec.CommandContext(ctx, "go", "build", "-o", binPath, "./internal/harness/"+harness)
	build.Dir = repoRoot()
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		return fmt.Errorf("build: %w", err)
	}

	cmd := exec.CommandContext(ctx, binPath, "-provider", provider)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = localRPCEnv(os.Environ())
	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("timed out after %s", timeout)
		}
		return err
	}
	return nil
}
