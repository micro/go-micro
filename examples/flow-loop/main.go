// Agentic Loop — keep working until the goal is met, with a guaranteed ceiling
//
// The "loop" pattern from agentic AI: instead of one shot, run a step over
// and over until the goal is reached, letting it decide when to stop — but
// always bounded by a hard iteration cap (the guardrail) so it can never run
// away, or run up an unbounded bill.
//
// flow.Loop is just a flow step, so it composes with the normal checkpointed
// step engine. This example needs no LLM key: the body is a plain func that
// "improves a draft" each pass, and a code-defined Until stops it once the
// draft is good enough — capped by FlowLoopMax. In a real flow the body would
// be micro.FlowDispatch("coder") (an agent) or micro.FlowLLM(...), and the
// stop check micro.FlowUntilLLM("Is the work complete?") — the supervised
// "Ralph" loop, where the model decides it's done but the cap still bounds it.
package main

import (
	"context"
	"fmt"

	"go-micro.dev/v6"
)

// Draft is the payload carried across iterations via State.Set / State.Scan.
type Draft struct {
	Text    string `json:"text"`
	Quality int    `json:"quality"` // 0..100, improved each pass
}

// improve is one loop pass: it refines the draft a bit. In a real flow this
// would be an agent or an LLM turn; here it's deterministic so the example
// runs offline.
func improve(_ context.Context, in micro.FlowState) (micro.FlowState, error) {
	var d Draft
	_ = in.Scan(&d)
	d.Quality += 30
	d.Text = fmt.Sprintf("draft refined (quality %d)", d.Quality)
	return in, in.Set(d)
}

func main() {
	const goodEnough = 90

	f := micro.NewFlow("refine",
		micro.FlowSteps(
			micro.FlowStep{Name: "improve", Run: micro.FlowLoop(
				improve,
				// Stop early once the draft is good enough...
				micro.FlowUntil(func(_ context.Context, s micro.FlowState, iter int) (bool, error) {
					var d Draft
					_ = s.Scan(&d)
					fmt.Printf("  pass %d → quality %d\n", iter, d.Quality)
					return d.Quality >= goodEnough, nil
				}),
				// ...but never run the body more than 10 times (the ceiling).
				micro.FlowLoopMax(10),
			)},
		),
		micro.FlowDeleteOnSuccess(),
	)

	fmt.Println("refining until quality >=", goodEnough)
	if err := f.Execute(context.Background(), `{"text":"initial draft","quality":0}`); err != nil {
		fmt.Println("flow error:", err)
		return
	}

	for _, r := range f.Results() {
		fmt.Printf("\ndone: %s\n", r.Answer)
	}
}
