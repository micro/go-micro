package flow

import (
	"errors"
	"fmt"

	hdag "github.com/heimdalr/dag"
)

var (
	errVertex = errors.New("vertex must have Name() method")
)

type dag interface {
	AddVertex(*Step) error
	AddEdge(*Step, *Step) error
	GetRoot() (*Step, error)
	GetVertex(string) (*Step, error)
	OrderedDescendants(*Step) ([]*Step, error)
	OrderedAncestors(*Step) ([]*Step, error)
	Validate() error
}

type heimdalrDag struct {
	dag *hdag.DAG
}

func newHeimdalrDag() *heimdalrDag {
	return &heimdalrDag{
		dag: hdag.NewDAG(),
	}
}

func (g *heimdalrDag) AddVertex(v *Step) error {
	return g.dag.AddVertex(v)
}

func (g *heimdalrDag) AddEdge(src *Step, dst *Step) error {
	return g.dag.AddEdge(src, dst)
}

func (g *heimdalrDag) OrderedAncestors(v *Step) ([]*Step, error) {
	hvcs, err := g.dag.GetOrderedAncestors(v)
	if err != nil {
		return nil, err
	}

	vcs := make([]*Step, 0, len(hvcs))
	vcs = append(vcs, v)
	for _, hv := range hvcs {
		vcs = append(vcs, hv.(*Step))
	}

	return vcs, nil
}

func (g *heimdalrDag) OrderedDescendants(v *Step) ([]*Step, error) {
	hvcs, err := g.dag.GetOrderedDescendants(v)
	if err != nil {
		return nil, err
	}

	vcs := make([]*Step, 0, len(hvcs))
	vcs = append(vcs, v)
	for _, hv := range hvcs {
		vcs = append(vcs, hv.(*Step))
	}

	return vcs, nil
}

func (g *heimdalrDag) Validate() error {
	return nil
}

func (g *heimdalrDag) TransitiveReduction() {
	g.dag.ReduceTransitively()
}

func (g *heimdalrDag) GetVertex(name string) (*Step, error) {
	v, err := g.dag.GetVertex(name)
	if err != nil {
		return nil, fmt.Errorf("step %s not found", name)
	}

	return v.(*Step), nil
}

func (g *heimdalrDag) GetRoot() (*Step, error) {
	roots := g.dag.GetRoots()
	if len(roots) != 1 {
		return nil, fmt.Errorf("dag have no or multiple roots")
	}
	for v, _ := range roots {
		return v.(*Step), nil
	}
	return nil, fmt.Errorf("dag have no or multiple roots")
}
