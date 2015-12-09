package registry

import (
	"testing"
)

func TestRoundRobinSelector(t *testing.T) {
	counts := map[string]int{}

	rr := &roundRobinSelector{
		so: SelectorOptions{
			Registry: &mockRegistry{},
		},
	}

	next, err := rr.Select("foo")
	if err != nil {
		t.Errorf("Unexpected error calling rr select: %v", err)
	}

	for i := 0; i < 100; i++ {
		node, err := next()
		if err != nil {
			t.Errorf("Expected node err, got err: %v", err)
		}
		counts[node.Id]++
	}

	t.Logf("Round Robin Counts %v", counts)
}
