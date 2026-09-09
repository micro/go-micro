package flow_test

import (
	"context"
	"fmt"
	"strings"

	"go-micro.dev/v6/flow"
)

func ExampleVerify() {
	generate := func(_ context.Context, in flow.State) (flow.State, error) {
		if strings.Contains(in.String(), "feedback") {
			in.Data = []byte(`{"answer":"include a source"}`)
			return in, nil
		}
		in.Data = []byte(`{"answer":"draft"}`)
		return in, nil
	}
	grader := func(_ context.Context, out flow.State) (bool, string, error) {
		return strings.Contains(out.String(), "source"), "add a source", nil
	}

	out, _ := flow.Verify(generate, grader, flow.VerifyMaxAttempts(2))(context.Background(), flow.State{})
	fmt.Println(strings.Contains(out.String(), `"verification_passed":true`))
	// Output: true
}

func ExampleAnalyze() {
	runs := []flow.Run{{
		ID: "run-1",
		Steps: []flow.StepRecord{{
			Name:   "draft",
			Status: "done",
			Result: `{"verification_passed":false,"verification_feedback":"add a source"}`,
		}},
	}}

	report := flow.Analyze(runs)
	fmt.Println(report.Candidates[0].Step)
	fmt.Println(report.Candidates[0].SampleFeedback[0])
	// Output:
	// draft
	// add a source
}
