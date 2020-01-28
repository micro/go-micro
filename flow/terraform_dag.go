package flow

import (
	"fmt"
	"sort"

	hdag "github.com/hashicorp/terraform/dag"
)

type walkerStep struct {
	step *Step
	pos  int
}

type walker struct {
	steps []walkerStep
}

func (w *walker) Walk(n hdag.Vertex, pos int) error {
	w.steps = append(w.steps, walkerStep{step: n.(*Step), pos: pos})
	return nil
}

type terraformDag struct {
	dag   *hdag.AcyclicGraph
	names map[string]interface{}
}

func NewTerraformDag() *terraformDag {
	return &terraformDag{
		dag:   &hdag.AcyclicGraph{},
		names: make(map[string]interface{}),
	}
}

func (g *terraformDag) AddVertex(v interface{}) error {
	vn, ok := v.(hdag.NamedVertex)
	if !ok {
		return fmt.Errorf("vertex have no Name() method")
	}
	g.names[vn.Name()] = v
	g.dag.Add(v)
	return nil
}

func (g *terraformDag) AddEdge(src interface{}, dst interface{}) error {
	g.dag.Connect(hdag.BasicEdge(src, dst))
	return nil
}

func (g *terraformDag) OrderedAncestors(v interface{}) ([]*Step, error) {
	w := &walker{}
	err := g.dag.DepthFirstWalk([]hdag.Vertex{v}, w.Walk)
	if err != nil {
		return nil, err
	}

	// sort steps for forward execution
	sort.Slice(w.steps, func(i, j int) bool {
		return w.steps[i].pos > w.steps[j].pos
	})

	steps := make([]*Step, 0, len(w.steps))
	for _, wstep := range w.steps {
		steps = append(steps, wstep.step)
	}

	return steps, nil
}

func (g *terraformDag) OrderedDescendants(v interface{}) ([]*Step, error) {
	w := &walker{}
	err := g.dag.DepthFirstWalk([]hdag.Vertex{v}, w.Walk)
	if err != nil {
		return nil, err
	}

	// sort steps for forward execution
	sort.Slice(w.steps, func(i, j int) bool {
		return w.steps[i].pos < w.steps[j].pos
	})

	steps := make([]*Step, 0, len(w.steps))
	for _, wstep := range w.steps {
		steps = append(steps, wstep.step)
	}

	return steps, nil
}

func (g *terraformDag) Validate() error {
	return g.dag.Validate()
}

func (g *terraformDag) TransitiveReduction() {
	g.dag.TransitiveReduction()
}

func (g *terraformDag) GetVertex(name string) (interface{}, error) {
	v, ok := g.names[name]
	if !ok {
		return nil, fmt.Errorf("vertex not found")
	}
	return v, nil
}
