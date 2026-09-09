package flow

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go-micro.dev/v6/ai"
)

// Grader checks a step output against a rubric. It returns pass=true when the
// output is acceptable; otherwise feedback should explain what the next attempt
// should fix.
type Grader func(ctx context.Context, out State) (pass bool, feedback string, err error)

// VerifyOptions configure Verify.
type VerifyOptions struct {
	// MaxAttempts bounds how many times the body can run. Default 2.
	MaxAttempts int
	// Backoff waits between failed grades. Zero means retry immediately.
	Backoff time.Duration
	// FeedbackField is the JSON field used to thread grader feedback into the
	// next attempt's input. Default "feedback".
	FeedbackField string
}

// VerifyOption configures Verify.
type VerifyOption func(*VerifyOptions)

// VerifyMaxAttempts sets the total attempt budget for Verify. Values <= 0 use
// the default of 2.
func VerifyMaxAttempts(n int) VerifyOption { return func(o *VerifyOptions) { o.MaxAttempts = n } }

// VerifyBackoff sets the delay between failed verification attempts.
func VerifyBackoff(d time.Duration) VerifyOption { return func(o *VerifyOptions) { o.Backoff = d } }

// VerifyFeedbackField sets the JSON field used to pass grader feedback to the
// next body attempt. Empty values use "feedback".
func VerifyFeedbackField(field string) VerifyOption {
	return func(o *VerifyOptions) { o.FeedbackField = field }
}

// Verify runs body, grades its output, and retries with grader feedback threaded
// into the next input until the grader passes or MaxAttempts is exhausted. It is
// a StepFunc, so it composes directly as Step.Run with Loop, LLM, Call, Agent, or
// any code-defined step.
//
// On a failed grade, Verify adds the feedback to the next attempt's input as a
// JSON field named "feedback" (or VerifyFeedbackField). When all attempts fail,
// it returns the last output without error, annotated with verification fields so
// the run can keep the bounded failure outcome in its state:
// "verification_passed": false, "verification_feedback", and
// "verification_attempts".
func Verify(body StepFunc, grader Grader, opts ...VerifyOption) StepFunc {
	o := VerifyOptions{MaxAttempts: 2, FeedbackField: "feedback"}
	for _, op := range opts {
		op(&o)
	}
	if o.MaxAttempts <= 0 {
		o.MaxAttempts = 2
	}
	if o.FeedbackField == "" {
		o.FeedbackField = "feedback"
	}
	return func(ctx context.Context, in State) (State, error) {
		if body == nil {
			return in, fmt.Errorf("flow: Verify requires a body step")
		}
		if grader == nil {
			return in, fmt.Errorf("flow: Verify requires a grader")
		}
		cur := in
		last := in
		feedback := ""
		for attempt := 1; attempt <= o.MaxAttempts; attempt++ {
			if err := ctx.Err(); err != nil {
				return last, err
			}
			if feedback != "" {
				var err error
				cur, err = stateWithField(cur, o.FeedbackField, feedback)
				if err != nil {
					return last, err
				}
			}
			out, err := body(ctx, cur)
			if err != nil {
				return last, fmt.Errorf("verify attempt %d: %w", attempt, err)
			}
			last = out
			pass, fb, err := grader(ctx, out)
			if err != nil {
				return last, fmt.Errorf("verify grade attempt %d: %w", attempt, err)
			}
			if pass {
				return stateWithVerification(out, true, fb, attempt)
			}
			feedback = fb
			cur = in
			if attempt < o.MaxAttempts && o.Backoff > 0 {
				select {
				case <-time.After(o.Backoff):
				case <-ctx.Done():
					return last, ctx.Err()
				}
			}
		}
		return stateWithVerification(last, false, feedback, o.MaxAttempts)
	}
}

// LLMGrader returns a grader that asks the flow model to judge the latest output
// against rubric. The model should answer with pass/fail plus short feedback.
// It reuses the flow's configured model, so it must run inside a flow.
func LLMGrader(rubric string) Grader {
	return func(ctx context.Context, out State) (bool, string, error) {
		d := depsFrom(ctx)
		if d == nil || d.model == nil {
			return false, "", fmt.Errorf("flow: LLMGrader requires a flow model (set Provider/APIKey)")
		}
		prompt := fmt.Sprintf("Grade the latest result against this rubric:\n%s\n\nLatest result:\n%s\n\nAnswer with PASS or FAIL on the first line, followed by one short feedback sentence.", rubric, out.String())
		resp, err := d.model.Generate(ctx, &ai.Request{Prompt: prompt})
		if err != nil {
			return false, "", err
		}
		reply := resp.Answer
		if reply == "" {
			reply = resp.Reply
		}
		return parseGrade(reply)
	}
}

func parseGrade(reply string) (bool, string, error) {
	text := strings.TrimSpace(reply)
	if text == "" {
		return false, "", fmt.Errorf("flow: LLMGrader returned an empty grade")
	}
	lines := strings.SplitN(text, "\n", 2)
	first := strings.ToLower(strings.TrimSpace(lines[0]))
	feedback := ""
	if len(lines) > 1 {
		feedback = strings.TrimSpace(lines[1])
	}
	pass := strings.HasPrefix(first, "pass") || isAffirmative(first)
	if !pass && feedback == "" {
		feedback = text
	}
	return pass, feedback, nil
}

func stateWithField(s State, field, value string) (State, error) {
	var obj map[string]any
	if len(s.Data) > 0 && json.Unmarshal(s.Data, &obj) == nil && obj != nil {
		obj[field] = value
		return stateWithObject(s, obj)
	}
	obj = map[string]any{field: value}
	if len(s.Data) > 0 {
		obj["data"] = s.String()
	}
	return stateWithObject(s, obj)
}

func stateWithVerification(s State, passed bool, feedback string, attempts int) (State, error) {
	var obj map[string]any
	if len(s.Data) > 0 && json.Unmarshal(s.Data, &obj) == nil && obj != nil {
		obj["verification_passed"] = passed
		obj["verification_feedback"] = feedback
		obj["verification_attempts"] = attempts
		return stateWithObject(s, obj)
	}
	obj = map[string]any{
		"data":                  s.String(),
		"verification_passed":   passed,
		"verification_feedback": feedback,
		"verification_attempts": attempts,
	}
	return stateWithObject(s, obj)
}

func stateWithObject(s State, obj map[string]any) (State, error) {
	b, err := json.Marshal(obj)
	if err != nil {
		return s, err
	}
	s.Data = b
	return s, nil
}
