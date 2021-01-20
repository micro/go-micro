package selector

import (
	"os"
	"testing"

	"github.com/asim/go-micro/v3/registry"
)

func TestRegistrySelector(t *testing.T) {
	counts := map[string]int{}

	r := registry.NewMemoryRegistry(registry.Services(testData))
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

	if len(os.Getenv("IN_TRAVIS_CI")) == 0 {
		t.Logf("Selector Counts %v", counts)
	}
}
