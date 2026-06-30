package flow

import (
	"context"
	"strings"
	"testing"
	"time"

	"go-micro.dev/v6/ai"
)

func TestAnalyzeRanksFailedGraderStepAbovePassingStep(t *testing.T) {
	now := time.Now()
	runs := []Run{
		{ID: "run-1", Started: now, Updated: now.Add(time.Second), Steps: []StepRecord{
			{Name: "draft", Status: "done", Attempts: 2, Result: `{"verification_passed":false,"verification_feedback":"cite sources"}`},
			{Name: "publish", Status: "done", Attempts: 1, Result: `{"verification_passed":true,"verification_feedback":"ok"}`},
		}},
		{ID: "run-2", Started: now, Updated: now.Add(2 * time.Second), Steps: []StepRecord{
			{Name: "draft", Status: "done", Attempts: 1, Result: `{"verification_passed":false,"verification_feedback":"too vague"}`},
			{Name: "publish", Status: "done", Attempts: 1, Result: `{"verification_passed":true,"verification_feedback":"ok"}`},
		}},
	}

	report := Analyze(runs)
	if len(report.Candidates) != 2 {
		t.Fatalf("Analyze returned %d candidates, want 2", len(report.Candidates))
	}
	if got := report.Candidates[0].Step; got != "draft" {
		t.Fatalf("top candidate = %q, want draft", got)
	}
	if report.Candidates[0].PassRate != 0 {
		t.Fatalf("draft pass rate = %v, want 0", report.Candidates[0].PassRate)
	}
}

func TestAnalyzeCarriesFeedbackSamplesAndRunIDs(t *testing.T) {
	report := Analyze([]Run{{ID: "run-9", Steps: []StepRecord{{
		Name: "grade", Status: "done", Attempts: 3,
		Result: `{"verification_passed":false,"verification_feedback":"include totals"}`,
	}}}})
	if len(report.Candidates) != 1 {
		t.Fatalf("candidates = %d, want 1", len(report.Candidates))
	}
	c := report.Candidates[0]
	if len(c.SampleFeedback) != 1 || c.SampleFeedback[0] != "include totals" {
		t.Fatalf("feedback = %#v, want include totals", c.SampleFeedback)
	}
	if len(c.RunIDs) != 1 || c.RunIDs[0] != "run-9" {
		t.Fatalf("run ids = %#v, want run-9", c.RunIDs)
	}
	if c.AverageRetries != 2 {
		t.Fatalf("average retries = %v, want 2", c.AverageRetries)
	}
}

func TestAnalyzeEmptyWindowReturnsEmptyReport(t *testing.T) {
	if got := Analyze(nil); len(got.Candidates) != 0 {
		t.Fatalf("empty Analyze candidates = %d, want 0", len(got.Candidates))
	}
}

func TestLLMOptimizerReturnsProposalWithoutMutatingFlow(t *testing.T) {
	f := New("optimize", Prompt("original prompt"))
	before := f.opts.Prompt
	optimizer := LLMOptimizer(&optimizerModel{reply: "revised prompt"})
	proposal, err := optimizer.OptimizePrompt(context.Background(), Candidate{Step: "draft", Metric: "pass_rate", SampleFeedback: []string{"cite sources"}}, before)
	if err != nil {
		t.Fatalf("OptimizePrompt returned error: %v", err)
	}
	if !strings.Contains(proposal, "revised") {
		t.Fatalf("proposal = %q, want revised prompt", proposal)
	}
	if f.opts.Prompt != before {
		t.Fatalf("flow prompt mutated to %q, want %q", f.opts.Prompt, before)
	}
}

type optimizerModel struct{ reply string }

func (m *optimizerModel) Init(...ai.Option) error { return nil }
func (m *optimizerModel) Options() ai.Options     { return ai.Options{} }
func (m *optimizerModel) Generate(context.Context, *ai.Request, ...ai.GenerateOption) (*ai.Response, error) {
	return &ai.Response{Reply: m.reply}, nil
}
func (m *optimizerModel) Stream(context.Context, *ai.Request, ...ai.GenerateOption) (ai.Stream, error) {
	return nil, ai.ErrStreamingUnsupported
}
func (m *optimizerModel) String() string { return "optimizer" }
