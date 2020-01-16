package flow_test

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/hashicorp/terraform/dag"
	flow "github.com/micro/go-micro/flow"
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

	sStore := memory.NewDataStore()
	dStore := memory.NewDataStore()
	fStore := memory.NewFlowStore()

	fl := flow.NewFlow(
		flow.WithStateStore(sStore),
		flow.WithDataStore(dStore),
		flow.WithFlowStore(fStore),
	)

	if err = fl.Init(); err != nil {
		t.Fatal(err)
	}
	if err = fl.CreateStep(ctx, "forward", &flow.Step{
		Name:       "AccountCreate",
		Operations: []flow.Operation{flow.ClientCallOperation("cms_account", "AccountService.AccountCreate")},
		Requires:   nil,
		Required:   nil,
	}); err != nil {
		t.Fatal(err)
	}
	if err = fl.CreateStep(ctx, "forward", &flow.Step{
		Name:       "ContactCreate",
		Operations: []flow.Operation{flow.ClientCallOperation("cms_contact", "ContactService.ContactCreate")},
		Requires:   []string{"AccountCreate"},
		Required:   nil,
	}); err != nil {
		t.Fatal(err)
	}
	if err = fl.CreateStep(ctx, "forward", &flow.Step{
		Name:       "ProjectCreate",
		Operations: []flow.Operation{flow.ClientCallOperation("cms_project", "ProjectService.ProjectCreate")},
		Requires:   []string{"AccountCreate"},
		Required:   nil,
	}); err != nil {
		t.Fatal(err)
	}
	if err = fl.CreateStep(ctx, "forward", &flow.Step{
		Name:       "NetworkCreate",
		Operations: []flow.Operation{flow.ClientCallOperation("cms_network", "NetworkService.NetworkCreate")},
		Requires:   []string{"AccountCreate"},
		Required:   nil,
	}); err != nil {
		t.Fatal(err)
	}
	if err = fl.CreateStep(ctx, "forward", &flow.Step{
		Name:       "AuthzCreate",
		Operations: []flow.Operation{flow.ClientCallOperation("cms_authz", "AuthzService.AuthzCreate")},
		Requires:   []string{"AccountCreate"},
		Required:   nil,
	}); err != nil {
		t.Fatal(err)
	}
	if err = fl.CreateStep(ctx, "forward", &flow.Step{
		Name:       "MailSend",
		Operations: []flow.Operation{flow.ClientCallOperation("cms_mailer", "MailService.MailSend")},
		Requires:   []string{"all"},
		Required:   nil,
	}); err != nil {
		t.Fatal(err)
	}

	req := &proto.Test{Name: "req"}
	rsp := &proto.Test{}

	//	err  = fl.
	rid, err := fl.Execute(ctx, "forward", req, rsp, flow.ExecuteAsync(false))
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("rid %s", rid)
}

func TestDag(t *testing.T) {
	t.Skip()
	g := &dag.AcyclicGraph{}

	n1 := g.Add(flow.ClientCallOperation("cms_account", "AccountService.AccountCreate"))
	n2 := g.Add(flow.ClientCallOperation("cms_contact", "ContactService.ContactCreate"))
	n3 := g.Add(flow.ClientCallOperation("cms_project", "ProjectService.ProjectCreate"))
	n4 := g.Add(flow.ClientCallOperation("cms_network", "NetworkService.NetworkCreate"))
	n5 := g.Add(flow.ClientCallOperation("cms_authz", "AuthzService.AuthzCreate"))
	n6 := g.Add(flow.ClientCallOperation("cms_mailer", "MailerService.MailSend"))

	g.Connect(dag.BasicEdge(n1, n2))
	g.Connect(dag.BasicEdge(n1, n3))
	g.Connect(dag.BasicEdge(n1, n4))
	g.Connect(dag.BasicEdge(n1, n5))
	g.Connect(dag.BasicEdge(n1, n6))

	g.Connect(dag.BasicEdge(n2, n6))
	g.Connect(dag.BasicEdge(n3, n6))
	g.Connect(dag.BasicEdge(n4, n6))
	g.Connect(dag.BasicEdge(n5, n6))

	if err := g.Validate(); err != nil {
		t.Fatal(err)
	}

	g.TransitiveReduction()

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

	vs, err := getVertex(g, flow.ClientCallOperation("cms_mailer", "MailerService.MailSend").String())
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
