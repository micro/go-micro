package label

import (
	"testing"

	"github.com/asim/go-micro/v3/selector"
	"github.com/asim/go-micro/v3/registry"
	"github.com/asim/go-micro/plugins/registry/memory/v3"
)

func TestPrioritiseFunc(t *testing.T) {
	nodes := []*registry.Node{
		&registry.Node{
			Id: "1",
			Metadata: map[string]string{
				"key1": "val1",
			},
		},
		&registry.Node{
			Id: "2",
			Metadata: map[string]string{
				"key2": "val2",
			},
		},
		&registry.Node{
			Id: "3",
			Metadata: map[string]string{
				"key1": "val1",
			},
		},
		&registry.Node{
			Id: "4",
		},
	}

	labels := []label{
		label{"key2", "val2"},
	}

	lnodes := prioritise(nodes, labels)
	t.Log("Prioritised node list #1")
	for _, node := range lnodes {
		t.Logf("Node %+v", node)
	}

	if id := lnodes[0].Id; id != "2" {
		t.Errorf("Expected node with id 2, got id: %s", id)
	}

	labels = []label{
		label{"key1", "val1"},
		label{"key2", "val2"},
	}

	lnodes = prioritise(nodes, labels)
	t.Log("Prioritised node list #2")
	for _, node := range lnodes {
		t.Logf("Node %+v", node)
	}

	data := []struct {
		i  int
		id string
	}{
		{0, "1"},
		{1, "3"},
		{2, "2"},
	}

	for _, d := range data {
		if id := lnodes[d.i].Id; id != d.id {
			t.Errorf("Expected node with id %s, got id: %s", d.id, id)
		}
	}
}

func TestLabelSelector(t *testing.T) {
	counts := map[string]int{}

	r := memory.NewRegistry()
	r.Register(&registry.Service{
		Name:    "bar",
		Version: "latest",
		Nodes: []*registry.Node{
			&registry.Node{
				Id: "1",
				Metadata: map[string]string{
					"key1": "val1",
				},
			},
			&registry.Node{
				Id: "2",
				Metadata: map[string]string{
					"key2": "val2",
				},
			},
		},
	})

	r.Register(&registry.Service{
		Name:    "bar",
		Version: "1.0.0",
		Nodes: []*registry.Node{
			&registry.Node{
				Id: "3",
				Metadata: map[string]string{
					"key1": "val1",
				},
			},
			&registry.Node{
				Id: "4",
			},
		},
	})

	ls := NewSelector(
		selector.Registry(r),
		Label("key2", "val2"),
		Label("key1", "val1"),
	)

	next, err := ls.Select("bar")
	if err != nil {
		t.Errorf("Unexpected error calling ls select: %v", err)
	}

	for i := 0; i < 100; i++ {
		node, err := next()
		if err != nil {
			t.Errorf("Expected node err, got err: %v", err)
		}
		counts[node.Id]++
	}

	t.Logf("Label Select Counts %v", counts)
}
