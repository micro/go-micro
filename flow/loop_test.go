package flow

import (
	"context"
	"strconv"
	"testing"
)

// counter body: increments an integer carried in State.Data.
func counter() StepFunc {
	return func(ctx context.Context, in State) (State, error) {
		n, _ := strconv.Atoi(in.String())
		in.Data = []byte(strconv.Itoa(n + 1))
		return in, nil
	}
}

func TestLoopUntil(t *testing.T) {
	step := Loop(counter(),
		Until(func(ctx context.Context, s State, iter int) (bool, error) {
			n, _ := strconv.Atoi(s.String())
			return n >= 3, nil
		}),
		LoopMax(100),
	)
	out, err := step(context.Background(), State{Data: []byte("0")})
	if err != nil {
		t.Fatal(err)
	}
	if out.String() != "3" {
		t.Fatalf("expected 3, got %q", out.String())
	}
}

func TestLoopMaxCapStops(t *testing.T) {
	runs := 0
	body := func(ctx context.Context, in State) (State, error) { runs++; return in, nil }
	// condition never fires; the cap must stop it
	step := Loop(body,
		Until(func(ctx context.Context, s State, iter int) (bool, error) { return false, nil }),
		LoopMax(5),
	)
	if _, err := step(context.Background(), State{}); err != nil {
		t.Fatal(err)
	}
	if runs != 5 {
		t.Fatalf("expected 5 iterations (cap), got %d", runs)
	}
}

func TestLoopOnIteration(t *testing.T) {
	var seen []int
	body := func(ctx context.Context, in State) (State, error) { return in, nil }
	step := Loop(body, LoopMax(3), OnIteration(func(iter int, s State) { seen = append(seen, iter) }))
	if _, err := step(context.Background(), State{}); err != nil {
		t.Fatal(err)
	}
	if len(seen) != 3 || seen[0] != 1 || seen[2] != 3 {
		t.Fatalf("expected iterations [1 2 3], got %v", seen)
	}
}

func TestLoopBodyError(t *testing.T) {
	body := func(ctx context.Context, in State) (State, error) {
		return in, context.Canceled
	}
	step := Loop(body, LoopMax(3))
	if _, err := step(context.Background(), State{}); err == nil {
		t.Fatal("expected error from body to propagate")
	}
}

func TestIsAffirmative(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"yes", true},
		{"Yes, the goal is met.", true},
		{"DONE", true},
		{"complete", true},
		{"no", false},
		{"not yet", false},
		{"", false},
	}
	for _, c := range cases {
		if got := isAffirmative(c.in); got != c.want {
			t.Errorf("isAffirmative(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}
