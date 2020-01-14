package flow

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/hashicorp/terraform/dag"
	proto "github.com/micro/go-micro/flow/proto"
	memory "github.com/micro/go-micro/flow/store/memory"
)

type node struct {
	item interface{}
	pos  int
}

type walk struct {
	nodes []node
}

func (w *walk) Walk(n dag.Vertex, pos int) error {
	w.nodes = append(w.nodes, node{item: n, pos: pos})
	return nil
}

func getVertex(g *dag.AcyclicGraph, name string) (dag.Vertex, error) {
	for _, v := range g.Vertices() {
		n := v.(dag.NamedVertex)
		if n.Name() == name {
			return v, nil
		}
	}
	return nil, fmt.Errorf("vertex %s not found", name)
}

func TestExecutor(t *testing.T) {
	var err error
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	sStore := memory.DefaultStateStore()
	dStore := memory.DefaultDataStore()
	fStore := memory.DefaultFlowStore()

	mgr := NewManager(ManagerFlowStore(fStore))

	if err = mgr.Init(); err != nil {
		t.Fatal(err)
	}

	if err = mgr.Register(&Flow{Name: "forward"}); err != nil {
		t.Fatal(err)
	}

	exc := NewExecutor(
		ExecutorStateStore(sStore),
		ExecutorDataStore(dStore),
		ExecutorFlowStore(fStore),
	)

	if err = exc.Init(); err != nil {
		t.Fatal(err)
	}

	req := &proto.Test{Name: "req"}
	rsp := &proto.Test{}

	rid, err := exc.Execute(ctx, "forward", req, rsp, ExecuteAsync(false))
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("rid %s", rid)
}

func TestDag(t *testing.T) {
	t.Skip()
	g := &dag.AcyclicGraph{}
	n1 := g.Add(ClientCallOperation("cms_account", "AccountService.AccountCreate"))
	n2 := g.Add(ClientCallOperation("cms_contact", "ContactService.ContactCreate"))
	n3 := g.Add(ClientCallOperation("cms_project", "ProjectService.ProjectCreate"))
	n4 := g.Add(ClientCallOperation("cms_network", "NetworkService.NetworkCreate"))
	n5 := g.Add(ClientCallOperation("cms_authz", "AuthzService.AuthzCreate"))
	n6 := g.Add(AggregateOperation())
	n7 := g.Add(ClientCallOperation("cms_mailer", "MailerService.MailSend"))
	g.Connect(dag.BasicEdge(n1, n2))
	g.Connect(dag.BasicEdge(n1, n3))
	g.Connect(dag.BasicEdge(n1, n4))
	g.Connect(dag.BasicEdge(n1, n5))

	g.Connect(dag.BasicEdge(n2, n6))
	g.Connect(dag.BasicEdge(n3, n6))
	g.Connect(dag.BasicEdge(n4, n6))
	g.Connect(dag.BasicEdge(n5, n6))
	g.Connect(dag.BasicEdge(n6, n7))

	if err := g.Validate(); err != nil {
		t.Fatal(err)
	}

	r, err := g.Root()
	if err != nil {
		t.Fatal(err)
	}

	tr := &walk{}
	err = g.DepthFirstWalk([]dag.Vertex{r}, tr.Walk)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("original\n")
	for _, n := range tr.nodes {
		fmt.Printf("node: %v pos: %d\n", n.item, n.pos)
	}

	fmt.Printf("forward\n")
	sort.Slice(tr.nodes, func(i, j int) bool {
		return tr.nodes[i].pos < tr.nodes[j].pos
	})
	for _, n := range tr.nodes {
		fmt.Printf("node: %v pos: %d\n", n.item, n.pos)
	}

	fmt.Printf("backward\n")
	sort.Slice(tr.nodes, func(i, j int) bool {
		return tr.nodes[i].pos > tr.nodes[j].pos
	})
	for _, n := range tr.nodes {
		fmt.Printf("node: %v pos: %d\n", n.item, n.pos)
	}

	vs, err := getVertex(g, ClientCallOperation("cms_mailer", "MailerService.MailSend").String())
	if err != nil {
		t.Fatal(err)
	}
	tr = &walk{}
	err = g.ReverseDepthFirstWalk([]dag.Vertex{vs}, tr.Walk)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("backward after\n")
	sort.Slice(tr.nodes, func(i, j int) bool {
		return tr.nodes[i].pos < tr.nodes[j].pos
	})
	for _, n := range tr.nodes {
		fmt.Printf("node: %v pos: %d\n", n.item, n.pos)
	}

}
