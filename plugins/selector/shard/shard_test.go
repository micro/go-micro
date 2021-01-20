package shard_test

import (
	"testing"

	"github.com/asim/go-micro/v3/client"
	"github.com/asim/go-micro/v3/selector"
	"github.com/asim/go-micro/v3/registry"
	"github.com/asim/go-micro/plugins/selector/shard/v3"
)

func TestShard(t *testing.T) {
	type args struct {
		keys  []string
		nodes []*registry.Node
		count int
	}

	type test struct {
		name string
		args args
		want *registry.Node
		err  error
	}

	node1 := &registry.Node{
		Id: "1",
	}

	node2 := &registry.Node{
		Id: "2",
	}

	node3 := &registry.Node{
		Id: "3",
	}

	nodes := func(n ...*registry.Node) []*registry.Node {
		return n
	}

	tests := []test{
		{
			name: "test single",
			args: args{
				keys:  []string{"a"},
				nodes: nodes(node1),
				count: 1,
			},
			want: node1,
		},
		{
			name: "test two nodes",
			args: args{
				keys:  []string{"c"},
				nodes: nodes(node1, node2),
				count: 1,
			},
			want: node2,
		},
		{
			name: "test three nodes",
			args: args{
				keys:  []string{"b"},
				nodes: nodes(node1, node2, node3),
				count: 1,
			},
			want: node3,
		},
		{
			name: "test three nodes two params",
			args: args{
				keys:  []string{"a", "a"},
				nodes: nodes(node1, node2, node3),
				count: 1,
			},
			want: node1,
		},
		{
			name: "test three nodes two params two cycles",
			args: args{
				keys:  []string{"a", "a"},
				nodes: nodes(node1, node2, node3),
				count: 2,
			},
			want: node2,
		},
		{
			name: "test three nodes two params three cycles",
			args: args{
				keys:  []string{"a", "a"},
				nodes: nodes(node1, node2, node3),
				count: 3,
			},
			want: node3,
		},
		{
			name: "test three nodes two params four cycles",
			args: args{
				keys:  []string{"a", "a"},
				nodes: nodes(node1, node2, node3),
				count: 4,
			},
			want: nil,
			err:  selector.ErrNoneAvailable,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn := shard.Strategy(tt.args.keys...)
			next := getNext(fn, tt.args.nodes)

			if next == nil {
				t.Error("Shard() got nil next")
				return
			}

			var err error
			var got *registry.Node
			for i := 0; i < tt.args.count; i++ {
				got, err = next()
			}

			if got != tt.want {
				t.Errorf("Shard() = %v, want %v", got, tt.want)
			}

			if tt.err != nil {
				if err == nil || err.Error() != tt.err.Error() {
					t.Errorf("Shard() error = %v, want %v", err, tt.err)
				}
			} else if err != nil {
				t.Errorf("Shard() unexpected error = %v", err)
			}
		})
	}
}

func getNext(fn client.CallOption, nodes []*registry.Node) selector.Next {
	co := &client.CallOptions{}
	fn(co)

	if len(co.SelectOptions) != 1 {
		return nil
	}

	opt := co.SelectOptions[0]

	so := &selector.SelectOptions{}
	opt(so)

	if so.Strategy == nil {
		return nil
	}

	return so.Strategy([]*registry.Service{
		{
			Nodes: nodes,
		},
	})
}
