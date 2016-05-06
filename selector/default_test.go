package selector

import (
	"errors"
	"testing"

	"github.com/micro/go-micro/registry"
	"github.com/micro/go-micro/registry/mock"
)

func TestDefaultSelector(t *testing.T) {
	counts := map[string]int{}

	rs := newDefaultSelector(Registry(mock.NewRegistry()))

	next, err := rs.Select("foo")
	if err != nil {
		t.Errorf("Unexpected error calling default select: %v", err)
	}

	for i := 0; i < 100; i++ {
		node, err := next()
		if err != nil {
			t.Errorf("Expected node err, got err: %v", err)
		}
		counts[node.Id]++
	}

	t.Logf("Default Counts %v", counts)
}

func TestBlackList(t *testing.T) {
	r := mock.NewRegistry()

	r.Register(&registry.Service{
		Name: "test",
		Nodes: []*registry.Node{
			&registry.Node{
				Id:      "test-1",
				Address: "localhost",
				Port:    10001,
			},
			&registry.Node{
				Id:      "test-2",
				Address: "localhost",
				Port:    10002,
			},
			&registry.Node{
				Id:      "test-3",
				Address: "localhost",
				Port:    10002,
			},
		},
	})

	rs := newDefaultSelector(Registry(r))

	next, err := rs.Select("test")
	if err != nil {
		t.Fatal(err)
	}

	node, err := next()
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 4; i++ {
		rs.Mark("test", node, errors.New("error"))
	}

	next, err = rs.Select("test")
	if err != nil {
		t.Fatal(err)
	}

	// still expecting 2 nodes
	seen := make(map[string]bool)

	for i := 0; i < 10; i++ {
		node, err = next()
		if err != nil {
			t.Fatal(err)
		}
		seen[node.Id] = true
	}

	if len(seen) != 2 {
		t.Fatalf("Expected seen to be 2 %+v", seen)
	}

	// blacklist all of it
	for i := 0; i < 9; i++ {
		node, err = next()
		if err != nil {
			t.Fatal(err)
		}
		rs.Mark("test", node, errors.New("error"))
	}

	next, err = rs.Select("test")
	if err != ErrNoneAvailable {
		t.Fatalf("Expected %v got %v", ErrNoneAvailable, err)
	}

}
