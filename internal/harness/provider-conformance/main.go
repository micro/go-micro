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
	flag.Parse()

	providers := splitCSV(*providersFlag)
	harnesses := splitCSV(*harnessesFlag)

	var ran, skipped, failed int
	for _, provider := range providers {
		if provider != "mock" && providerKey(provider) == "" {
			fmt.Printf("- %s: skipped (set MICRO_AI_API_KEY or %s)\n", provider, providerEnv[provider])
			skipped++
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

	cmd := exec.CommandContext(ctx, "go", "run", "./internal/harness/"+harness, "-provider", provider)
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
