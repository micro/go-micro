package selector

import (
	"testing"

	"github.com/micro/go-micro/registry/mock"
)

func TestRandomSelector(t *testing.T) {
	counts := map[string]int{}

	rs := &randomSelector{
		so: Options{
			Registry: mock.NewRegistry(),
		},
	}

	next, err := rs.Select("foo")
	if err != nil {
		t.Errorf("Unexpected error calling random select: %v", err)
	}

	for i := 0; i < 100; i++ {
		node, err := next()
		if err != nil {
			t.Errorf("Expected node err, got err: %v", err)
		}
		counts[node.Id]++
	}

	t.Logf("Random Counts %v", counts)
}
