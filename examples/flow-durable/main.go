// Durable Flow — a workflow that survives a crash and resumes
//
// A flow can be an ordered list of steps (a task with stages) rather than
// a single LLM turn. Each step is checkpointed before and after through a
// pluggable Checkpoint (store-backed by default), so if the process dies
// mid-run, the run resumes at the step it stopped on — without re-running
// the steps that already completed (and already had their side effects).
//
// This example needs no LLM key. It runs a three-step checkout flow whose
// "charge" step fails the first time (a transient outage). The run is
// checkpointed as failed at that step; we then "recover" the dependency
// and Resume — and the already-completed "reserve" step does not run
// again. A real step would call a service (flow.Call), an agent
// (flow.Dispatch), or the model (flow.LLM); here they're plain funcs so
// the durability is the only thing on display.
package main

import (
	"context"
	"errors"
	"fmt"

	"go-micro.dev/v5"
)

// Order is the payload carried across steps via State.Set / State.Scan.
type Order struct {
	ID        string `json:"id"`
	Reserved  bool   `json:"reserved"`
	Charged   bool   `json:"charged"`
	Confirmed bool   `json:"confirmed"`
}

// charged toggles the transient failure: 0 = the payment dependency is
// down (first run), 1 = recovered (on resume).
var charged int

// reserveCalls proves the completed step is not re-run on resume.
var reserveCalls int

func reserve(_ context.Context, in micro.FlowState) (micro.FlowState, error) {
	reserveCalls++
	var o Order
	in.Scan(&o)
	o.ID = "order-1"
	o.Reserved = true
	fmt.Println("  reserve  → inventory reserved")
	return in, in.Set(o)
}

func charge(_ context.Context, in micro.FlowState) (micro.FlowState, error) {
	var o Order
	in.Scan(&o)
	if charged == 0 {
		fmt.Println("  charge   → payment dependency unavailable (crash)")
		return in, errors.New("payment gateway timeout")
	}
	o.Charged = true
	fmt.Println("  charge   → payment captured")
	return in, in.Set(o)
}

func confirm(_ context.Context, in micro.FlowState) (micro.FlowState, error) {
	var o Order
	in.Scan(&o)
	o.Confirmed = true
	fmt.Println("  confirm  → order confirmed")
	return in, in.Set(o)
}

func main() {
	f := micro.NewFlow("checkout",
		micro.FlowSteps(
			micro.FlowStep{Name: "reserve", Run: reserve},
			micro.FlowStep{Name: "charge", Run: charge},
			micro.FlowStep{Name: "confirm", Run: confirm},
		),
		// Durable by default; shown explicitly. Runs are namespaced under
		// the flow name ("flow/checkout/runs/..."), so this flow's state
		// doesn't share a keyspace with other flows. Point the default
		// store at Postgres or NATS KV to survive a real process restart.
		micro.FlowWithCheckpoint(micro.StoreCheckpoint(nil, "checkout")),
	)

	ctx := context.Background()

	fmt.Println("first run:")
	if err := f.Execute(ctx, `{}`); err != nil {
		fmt.Printf("  run failed: %v\n", err)
	}

	pending, _ := f.Pending(ctx)
	if len(pending) == 0 {
		fmt.Println("nothing pending — unexpected")
		return
	}
	run := pending[0]
	fmt.Printf("\ncheckpoint: run %s is at step %q (status %s)\n",
		run.ID[:8], run.State.Stage, run.Status)

	// The dependency recovers (or a new process picks the run up).
	charged = 1

	fmt.Println("\nresume:")
	if err := f.Resume(ctx, run.ID); err != nil {
		fmt.Printf("  resume failed: %v\n", err)
		return
	}

	fmt.Printf("\nreserve ran %d time(s) total — completed steps are not repeated on resume\n", reserveCalls)
	if pend, _ := f.Pending(ctx); len(pend) == 0 {
		fmt.Println("no pending runs — the workflow completed durably")
	}
}
