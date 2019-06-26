package selector

import (
	"testing"

	"github.com/micro/go-micro/registry/memory"
)

func TestRegistrySelector(t *testing.T) {
	counts := map[string]int{}

	r := memory.NewRegistry()
	rg := r.(*memory.Registry)
	rg.Services = testData
	cache := NewSelector(Registry(r))

	next, err := cache.Select("foo")
	if err != nil {
		t.Errorf("Unexpected error calling cache select: %v", err)
	}

	for i := 0; i < 100; i++ {
		node, err := next()
		if err != nil {
			t.Errorf("Expected node err, got err: %v", err)
		}
		counts[node.Id]++
	}

	t.Logf("Selector Counts %v", counts)
}
