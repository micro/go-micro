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
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"go-micro.dev/v6/ai"
	_ "go-micro.dev/v6/ai/anthropic"
	_ "go-micro.dev/v6/ai/atlascloud"
	_ "go-micro.dev/v6/ai/gemini"
	_ "go-micro.dev/v6/ai/groq"
	_ "go-micro.dev/v6/ai/minimax"
	_ "go-micro.dev/v6/ai/mistral"
	_ "go-micro.dev/v6/ai/openai"
	_ "go-micro.dev/v6/ai/together"
)

const defaultHarnesses = "agent,universe,agent-flow,plan-delegate,a2a-stream-fallback"

var harnessPhases = map[string]string{
	"agent":               "model call + tool call",
	"universe":            "service discovery + tool call",
	"agent-flow":          "workflow event + tool call",
	"plan-delegate":       "plan persistence + delegation + tool call",
	"a2a-stream-fallback": "streaming fallback + tool call",
}

var providerEnv = map[string]string{
	"anthropic":  "ANTHROPIC_API_KEY",
	"openai":     "OPENAI_API_KEY",
	"gemini":     "GEMINI_API_KEY",
	"groq":       "GROQ_API_KEY",
	"minimax":    "MINIMAX_API_KEY",
	"mistral":    "MISTRAL_API_KEY",
	"together":   "TOGETHER_API_KEY",
	"atlascloud": "ATLASCLOUD_API_KEY",
}

func main() {
	providersFlag := flag.String("providers", defaultProviders(), "comma-separated providers to check; use mock for deterministic local checks")
	harnessesFlag := flag.String("harnesses", defaultHarnesses, "comma-separated harness names under internal/harness; agent runs the provider tool-call conformance test")
	timeoutFlag := flag.Duration("timeout", 10*time.Minute, "timeout per provider/harness run")
	requireConfiguredFlag := flag.Bool("require-configured", false, "fail when a selected live provider is missing an API key")
	capabilitiesFlag := flag.Bool("capabilities", true, "print the registered provider capability matrix before running conformance")
	summaryJSONFlag := flag.String("summary-json", "", "write a machine-readable conformance summary to this path")
	summaryMarkdownFlag := flag.String("summary-markdown", "", "write a human-readable conformance summary to this path")
	capabilityMarkdownFlag := flag.String("capabilities-markdown", "", "write the registered provider capability matrix as a Markdown table")
	flag.Parse()

	providers := splitCSV(*providersFlag)
	harnesses := splitCSV(*harnessesFlag)
	if err := validateSelection(providers, harnesses); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	if *capabilitiesFlag {
		printCapabilityMatrix()
	}
	if *capabilityMarkdownFlag != "" {
		if err := writeCapabilityMarkdown(*capabilityMarkdownFlag, ai.CapabilityRows()); err != nil {
			fmt.Fprintf(os.Stderr, "write capabilities markdown: %v\n", err)
			os.Exit(1)
		}
	}

	var ran, skipped, failed int
	var results []conformanceResult
	for _, provider := range providers {
		if provider != "mock" && providerKey(provider) == "" {
			msg := fmt.Sprintf("set MICRO_AI_API_KEY or %s", providerEnv[provider])
			if *requireConfiguredFlag {
				fmt.Printf("FAIL %s: missing API key (%s)\n", provider, msg)
				failed++
				results = append(results, conformanceResult{Provider: provider, Status: statusFailed, Error: "missing API key: " + msg})
			} else {
				fmt.Printf("- %s: skipped (%s)\n", provider, msg)
				skipped++
				results = append(results, conformanceResult{Provider: provider, Status: statusSkipped, Error: msg})
			}
			continue
		}

		for _, harness := range harnesses {
			phase := harnessPhase(harness)
			fmt.Printf("\n==> %s / %s (%s)\n", provider, harness, phase)
			if err := runHarness(provider, harness, *timeoutFlag); err != nil {
				fmt.Printf("FAIL %s / %s (%s): %v\n", provider, harness, phase, err)
				failed++
				results = append(results, conformanceResult{Provider: provider, Harness: harness, Phase: phase, Status: statusFailed, Error: err.Error()})
				continue
			}
			ran++
			results = append(results, conformanceResult{Provider: provider, Harness: harness, Phase: phase, Status: statusPassed})
		}
	}

	fmt.Printf("\nprovider conformance: %d passed, %d skipped providers, %d failed\n", ran, skipped, failed)
	summary := conformanceSummary{
		Providers:    providers,
		Harnesses:    harnesses,
		Capabilities: ai.CapabilityRows(),
		Results:      results,
		Passed:       ran,
		Skipped:      skipped,
		Failed:       failed,
	}
	if *summaryJSONFlag != "" {
		if err := writeSummaryJSON(*summaryJSONFlag, summary); err != nil {
			fmt.Fprintf(os.Stderr, "write summary: %v\n", err)
			os.Exit(1)
		}
	}
	if *summaryMarkdownFlag != "" {
		if err := writeSummaryMarkdown(*summaryMarkdownFlag, summary); err != nil {
			fmt.Fprintf(os.Stderr, "write summary markdown: %v\n", err)
			os.Exit(1)
		}
	}
	if failed > 0 {
		os.Exit(1)
	}
}

const (
	statusPassed  = "passed"
	statusSkipped = "skipped"
	statusFailed  = "failed"
)

type conformanceResult struct {
	Provider string `json:"provider"`
	Harness  string `json:"harness,omitempty"`
	Phase    string `json:"phase,omitempty"`
	Status   string `json:"status"`
	Error    string `json:"error,omitempty"`
}

type conformanceSummary struct {
	Providers    []string            `json:"providers"`
	Harnesses    []string            `json:"harnesses"`
	Capabilities []ai.CapabilityRow  `json:"capabilities"`
	Results      []conformanceResult `json:"results"`
	Passed       int                 `json:"passed"`
	Skipped      int                 `json:"skipped"`
	Failed       int                 `json:"failed"`
}

func writeSummaryJSON(path string, summary conformanceSummary) error {
	b, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(path, b, 0o644)
}

func writeCapabilityMarkdown(path string, rows []ai.CapabilityRow) error {
	return os.WriteFile(path, []byte(capabilityMarkdown(rows)), 0o644)
}

func writeSummaryMarkdown(path string, summary conformanceSummary) error {
	var b strings.Builder
	b.WriteString("# Provider conformance summary\n\n")
	fmt.Fprintf(&b, "Passed: %d. Skipped providers: %d. Failed: %d.\n\n", summary.Passed, summary.Skipped, summary.Failed)
	fmt.Fprintf(&b, "Providers: %s.\n\n", markdownList(summary.Providers))
	fmt.Fprintf(&b, "Harnesses: %s.\n\n", markdownList(summary.Harnesses))
	b.WriteString("## Capability matrix\n\n")
	b.WriteString(capabilityMarkdown(summary.Capabilities))
	b.WriteString("\n## Harness results\n\n")
	b.WriteString("| Provider | Harness | Phase | Status | Detail |\n")
	b.WriteString("| --- | --- | --- | --- | --- |\n")
	for _, result := range summary.Results {
		harness := result.Harness
		if harness == "" {
			harness = "—"
		}
		phase := result.Phase
		if phase == "" {
			phase = "—"
		}
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n", result.Provider, harness, markdownCell(phase), result.Status, markdownCell(result.Error))
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func capabilityMarkdown(rows []ai.CapabilityRow) string {
	var b strings.Builder
	b.WriteString("| Provider | Model | Image | Video | Streaming |\n")
	b.WriteString("| --- | --- | --- | --- | --- |\n")
	for _, row := range rows {
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n", row.Provider, mark(row.Model), mark(row.Image), mark(row.Video), mark(row.Stream))
	}
	return b.String()
}

func markdownList(values []string) string {
	if len(values) == 0 {
		return "—"
	}
	escaped := make([]string, 0, len(values))
	for _, value := range values {
		escaped = append(escaped, "`"+strings.ReplaceAll(value, "`", "\\`")+"`")
	}
	return strings.Join(escaped, ", ")
}

func harnessPhase(harness string) string {
	if phase, ok := harnessPhases[harness]; ok {
		return phase
	}
	return "harness"
}

func markdownCell(s string) string {
	if s == "" {
		return "—"
	}
	s = strings.ReplaceAll(s, "|", "\\|")
	s = strings.ReplaceAll(s, "\n", "<br>")
	return s
}

func mark(ok bool) string {
	if ok {
		return "✅"
	}
	return "—"
}

func printCapabilityMatrix() {
	fmt.Println("Provider capability matrix:")
	fmt.Println("provider     model  image  video  stream")
	for _, row := range ai.CapabilityRows() {
		fmt.Printf("%-12s %-5s  %-5s  %-5s  %-6s\n", row.Provider, yesNo(row.Model), yesNo(row.Image), yesNo(row.Video), yesNo(row.Stream))
	}
	fmt.Println()
}

func yesNo(ok bool) string {
	if ok {
		return "yes"
	}
	return "no"
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
		if harness == "agent" {
			continue
		}
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

func defaultProviders() string {
	providers := make([]string, 0, len(providerEnv))
	for provider := range providerEnv {
		providers = append(providers, provider)
	}
	slices.Sort(providers)
	return strings.Join(providers, ",")
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
	if harness == "agent" {
		return runAgentConformance(provider, timeout)
	}

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

func runAgentConformance(provider string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	testProvider := provider
	if provider == "mock" {
		testProvider = "fake"
	}
	cmd := exec.CommandContext(ctx, "go", "test", "./agent", "-run", "TestAgentProvider(ConformanceMatrix|StreamConformanceMatrix)", "-count=1", "-v")
	cmd.Dir = repoRoot()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = localRPCEnv(append(os.Environ(),
		"GO_MICRO_AGENT_CONFORMANCE_LIVE=1",
		"GO_MICRO_AGENT_CONFORMANCE_PROVIDERS="+testProvider,
	))
	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("timed out after %s", timeout)
		}
		return err
	}
	return nil
}
