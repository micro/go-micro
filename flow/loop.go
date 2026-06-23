package flow

import (
	"context"
	"fmt"
	"strings"

	"go-micro.dev/v6/ai"
)

// LoopCondition decides whether a Loop should stop, given the latest state
// and the iteration just completed (1-based). Returning true ends the loop.
type LoopCondition func(ctx context.Context, state State, iter int) (bool, error)

// LoopOptions configure a Loop. Max is the hard iteration cap — the ceiling
// that guarantees the loop always terminates, however the stop is decided.
// Until and UntilLLM are the optional early-stop checks.
type LoopOptions struct {
	Max      int
	Until    LoopCondition
	UntilLLM string
	OnIter   func(iter int, state State)
}

// LoopOption configures a Loop.
type LoopOption func(*LoopOptions)

// LoopMax sets the hard iteration cap — the budget guardrail. The loop never
// runs the body more than n times, so it always terminates even when the
// stop condition never fires. Default 10.
func LoopMax(n int) LoopOption { return func(o *LoopOptions) { o.Max = n } }

// Until stops the loop when cond returns true after an iteration — a
// deterministic, code-defined exit condition.
func Until(cond LoopCondition) LoopOption { return func(o *LoopOptions) { o.Until = cond } }

// UntilLLM stops the loop when the flow's model judges the goal met. After
// each iteration it asks the model the question with the latest state and
// stops on an affirmative answer — the agent decides when it's done (the
// supervised "Ralph" loop), while LoopMax guarantees termination. Requires a
// flow model (set Provider/APIKey).
func UntilLLM(question string) LoopOption { return func(o *LoopOptions) { o.UntilLLM = question } }

// OnIteration runs fn after each iteration — useful for logging progress or
// persisting intermediate state.
func OnIteration(fn func(iter int, state State)) LoopOption {
	return func(o *LoopOptions) { o.OnIter = fn }
}

// Loop returns a StepFunc that runs body repeatedly until a stop condition is
// met or the iteration cap is reached, whichever comes first — the agentic
// "loop": keep working until the goal is done, with a guaranteed ceiling so
// it can never run away.
//
// Compose it as a flow step. The carried State flows from one pass to the
// next, so each iteration sees the previous result:
//
//	flow.New("refactor",
//	    flow.Provider("anthropic"),
//	    flow.Steps(
//	        flow.Step{Name: "improve", Run: flow.Loop(
//	            flow.Dispatch("coder"),
//	            flow.UntilLLM("Is the refactor complete with no duplicated abstractions left?"),
//	            flow.LoopMax(5),
//	        )},
//	    ),
//	)
//
// The loop runs as a single flow step: the flow checkpoints the loop's
// outcome, and a resume re-enters the step, so loop bodies should be safe to
// repeat. Use OnIteration to record per-pass progress. If the cap is hit
// before the stop condition fires, the loop returns the latest state rather
// than erroring — the guardrail did its job.
func Loop(body StepFunc, opts ...LoopOption) StepFunc {
	o := LoopOptions{Max: 10}
	for _, op := range opts {
		op(&o)
	}
	if o.Max <= 0 {
		o.Max = 10
	}
	return func(ctx context.Context, in State) (State, error) {
		if body == nil {
			return in, fmt.Errorf("flow: Loop requires a body step")
		}
		cur := in
		for iter := 1; iter <= o.Max; iter++ {
			out, err := body(ctx, cur)
			if err != nil {
				return cur, fmt.Errorf("loop iteration %d: %w", iter, err)
			}
			cur = out
			if o.OnIter != nil {
				o.OnIter(iter, cur)
			}
			done, err := loopDone(ctx, o, cur, iter)
			if err != nil {
				return cur, err
			}
			if done {
				return cur, nil
			}
		}
		return cur, nil
	}
}

// loopDone evaluates the stop conditions: a code-defined Until predicate
// and/or an LLM judgement. Either firing stops the loop.
func loopDone(ctx context.Context, o LoopOptions, state State, iter int) (bool, error) {
	if o.Until != nil {
		done, err := o.Until(ctx, state, iter)
		if err != nil || done {
			return done, err
		}
	}
	if o.UntilLLM != "" {
		return askDone(ctx, o.UntilLLM, state)
	}
	return false, nil
}

// askDone asks the flow model whether the goal is met given the current
// state, and returns true on an affirmative reply — the supervised stop check.
func askDone(ctx context.Context, question string, state State) (bool, error) {
	d := depsFrom(ctx)
	if d == nil || d.model == nil {
		return false, fmt.Errorf("flow: UntilLLM requires a flow model (set Provider/APIKey)")
	}
	prompt := fmt.Sprintf("%s\n\nLatest result:\n%s\n\nAnswer with only \"yes\" or \"no\".", question, state.String())
	resp, err := d.model.Generate(ctx, &ai.Request{Prompt: prompt})
	if err != nil {
		return false, err
	}
	reply := resp.Answer
	if reply == "" {
		reply = resp.Reply
	}
	return isAffirmative(reply), nil
}

// isAffirmative reports whether a model reply reads as "yes/done".
func isAffirmative(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	for _, p := range []string{"yes", "done", "true", "complete", "finished"} {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}
