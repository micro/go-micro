package flow

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"go-micro.dev/v6/ai"
)

// AnalyzeOptions configures Analyze.
type AnalyzeOptions struct {
	// MaxFeedbackSamples bounds the number of representative grader feedback
	// strings retained per candidate. Values <= 0 use a small default.
	MaxFeedbackSamples int
}

// AnalyzeOption configures Analyze.
type AnalyzeOption func(*AnalyzeOptions)

// AnalyzeMaxFeedbackSamples sets how many grader feedback examples are kept for
// each candidate in the report.
func AnalyzeMaxFeedbackSamples(n int) AnalyzeOption {
	return func(o *AnalyzeOptions) { o.MaxFeedbackSamples = n }
}

// Report is the machine-readable output of Analyze. Candidates are ordered from
// worst to best so an agent, CLI, or human can pick the first improvement to try.
type Report struct {
	Candidates []Candidate `json:"candidates"`
}

// Candidate identifies one underperforming flow step and the trace evidence that
// made it worth improving.
type Candidate struct {
	Step           string        `json:"step"`
	Metric         string        `json:"metric"`
	Score          float64       `json:"score"`
	Runs           int           `json:"runs"`
	Failures       int           `json:"failures"`
	PassRate       float64       `json:"pass_rate"`
	ErrorRate      float64       `json:"error_rate"`
	AverageRetries float64       `json:"average_retries"`
	P50Latency     time.Duration `json:"p50_latency"`
	P95Latency     time.Duration `json:"p95_latency"`
	SampleFeedback []string      `json:"sample_feedback,omitempty"`
	RunIDs         []string      `json:"run_ids,omitempty"`
}

// Analyze aggregates a bounded window of persisted flow runs and returns ranked
// hill-climbing candidates. It uses the same Run records read by Checkpoint.List:
// failed verification fields in step results drive pass-rate and feedback, step
// status drives error rate, and retry attempts contribute retry pressure. An
// empty window returns an empty report.
func Analyze(runs []Run, opts ...AnalyzeOption) Report {
	o := AnalyzeOptions{MaxFeedbackSamples: 3}
	for _, opt := range opts {
		opt(&o)
	}
	if o.MaxFeedbackSamples <= 0 {
		o.MaxFeedbackSamples = 3
	}

	stats := map[string]*stepStats{}
	for _, run := range runs {
		for _, step := range run.Steps {
			if step.Name == "" {
				continue
			}
			s := stats[step.Name]
			if s == nil {
				s = &stepStats{}
				stats[step.Name] = s
			}
			s.runs++
			s.runIDs = appendUnique(s.runIDs, run.ID)
			if step.Attempts > 1 {
				s.retries += step.Attempts - 1
			}
			if step.Status == "failed" || step.Error != "" {
				s.errors++
			}
			if len(run.Steps) > 0 && !run.Started.IsZero() && !run.Updated.IsZero() {
				s.latencies = append(s.latencies, run.Updated.Sub(run.Started)/time.Duration(len(run.Steps)))
			}
			passed, feedback, ok := verificationFields(step.Result)
			if ok {
				s.graded++
				if !passed {
					s.gradeFailures++
					if feedback != "" && len(s.feedback) < o.MaxFeedbackSamples {
						s.feedback = append(s.feedback, feedback)
					}
				}
			}
		}
	}

	report := Report{}
	for step, s := range stats {
		if s.runs == 0 {
			continue
		}
		failures := s.errors + s.gradeFailures
		passRate := 1.0
		if s.graded > 0 {
			passRate = float64(s.graded-s.gradeFailures) / float64(s.graded)
		} else if s.errors > 0 {
			passRate = float64(s.runs-s.errors) / float64(s.runs)
		}
		errorRate := float64(s.errors) / float64(s.runs)
		avgRetries := float64(s.retries) / float64(s.runs)
		score := float64(s.gradeFailures)*3 + float64(s.errors)*2 + avgRetries
		metric := "pass_rate"
		if s.gradeFailures == 0 && s.errors > 0 {
			metric = "error_rate"
		} else if s.gradeFailures == 0 && s.errors == 0 && s.retries > 0 {
			metric = "retry_count"
		}
		report.Candidates = append(report.Candidates, Candidate{
			Step: step, Metric: metric, Score: score, Runs: s.runs, Failures: failures,
			PassRate: passRate, ErrorRate: errorRate, AverageRetries: avgRetries,
			P50Latency: percentile(s.latencies, 0.50), P95Latency: percentile(s.latencies, 0.95),
			SampleFeedback: append([]string(nil), s.feedback...), RunIDs: append([]string(nil), s.runIDs...),
		})
	}
	sort.SliceStable(report.Candidates, func(i, j int) bool {
		a, b := report.Candidates[i], report.Candidates[j]
		if a.Score == b.Score {
			return a.Step < b.Step
		}
		return a.Score > b.Score
	})
	return report
}

type stepStats struct {
	runs, graded, gradeFailures, errors, retries int
	feedback, runIDs                             []string
	latencies                                    []time.Duration
}

// PromptOptimizer proposes prompt improvements for a candidate without mutating
// the source flow. Applying the returned prompt stays explicitly gated by the caller.
type PromptOptimizer struct{ model ai.Model }

// LLMOptimizer returns an optimizer that asks model to revise prompts for
// Analyze candidates. The model is injected so tests and callers can use mocks.
func LLMOptimizer(model ai.Model) *PromptOptimizer { return &PromptOptimizer{model: model} }

// OptimizePrompt asks the model for a revised prompt for candidate using the
// current prompt and trace feedback. It returns only the proposal; it never
// modifies a Flow, Step, or Checkpoint.
func (o *PromptOptimizer) OptimizePrompt(ctx context.Context, candidate Candidate, currentPrompt string) (string, error) {
	if o == nil || o.model == nil {
		return "", fmt.Errorf("flow: LLMOptimizer requires a model")
	}
	prompt := fmt.Sprintf("Revise this workflow step prompt to improve the failing step.\nStep: %s\nMetric: %s\nScore: %.2f\nFeedback:\n- %s\n\nCurrent prompt:\n%s\n\nReturn only the revised prompt.", candidate.Step, candidate.Metric, candidate.Score, strings.Join(candidate.SampleFeedback, "\n- "), currentPrompt)
	resp, err := o.model.Generate(ctx, &ai.Request{Prompt: prompt})
	if err != nil {
		return "", err
	}
	proposal := strings.TrimSpace(resp.Answer)
	if proposal == "" {
		proposal = strings.TrimSpace(resp.Reply)
	}
	if proposal == "" {
		return "", fmt.Errorf("flow: LLMOptimizer returned an empty prompt")
	}
	return proposal, nil
}

func verificationFields(result string) (bool, string, bool) {
	if result == "" {
		return false, "", false
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(result), &obj); err != nil {
		return false, "", false
	}
	v, ok := obj["verification_passed"].(bool)
	if !ok {
		return false, "", false
	}
	fb, _ := obj["verification_feedback"].(string)
	return v, fb, true
}

func appendUnique(values []string, value string) []string {
	if value == "" {
		return values
	}
	for _, v := range values {
		if v == value {
			return values
		}
	}
	return append(values, value)
}

func percentile(values []time.Duration, p float64) time.Duration {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]time.Duration(nil), values...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	idx := int(float64(len(sorted)-1) * p)
	return sorted[idx]
}
