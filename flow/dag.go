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
	AddVertex(interface{}) error
	AddEdge(interface{}, interface{}) error
	GetRoot() (interface{}, error)
	GetVertex(string) (interface{}, error)
	OrderedDescendants(interface{}) ([]*Step, error)
	OrderedAncestors(interface{}) ([]*Step, error)
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

func (g *heimdalrDag) AddVertex(v interface{}) error {
	vn, ok := v.(hdag.Vertex)
	if !ok {
		return errVertex
	}
	return g.dag.AddVertex(vn)
}

func (g *heimdalrDag) AddEdge(src interface{}, dst interface{}) error {
	vsrc, ok := src.(hdag.Vertex)
	if !ok {
		return errVertex
	}
	vdst, ok := dst.(hdag.Vertex)
	if !ok {
		return errVertex
	}

	return g.dag.AddEdge(vsrc, vdst)
}

func (g *heimdalrDag) OrderedAncestors(v interface{}) ([]*Step, error) {
	vn, ok := v.(hdag.Vertex)
	if !ok {
		return nil, errVertex
	}

	hvcs, err := g.dag.GetOrderedAncestors(vn)
	if err != nil {
		return nil, err
	}

	vcs := make([]*Step, 0, len(hvcs))
	vcs = append(vcs, v.(*Step))
	for _, hv := range hvcs {
		vcs = append(vcs, hv.(*Step))
	}

	return vcs, nil
}

func (g *heimdalrDag) OrderedDescendants(v interface{}) ([]*Step, error) {
	vn, ok := v.(hdag.Vertex)
	if !ok {
		return nil, errVertex
	}

	hvcs, err := g.dag.GetOrderedDescendants(vn)
	if err != nil {
		return nil, err
	}

	vcs := make([]*Step, 0, len(hvcs))
	vcs = append(vcs, v.(*Step))
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

func (g *heimdalrDag) GetVertex(name string) (interface{}, error) {
	return g.dag.GetVertex(name)
}

func (g *heimdalrDag) GetRoot() (interface{}, error) {
	roots := g.dag.GetRoots()
	if len(roots) != 1 {
		return nil, fmt.Errorf("dag have no or multiple roots")
	}
	for v, _ := range roots {
		return v, nil
	}
	return nil, fmt.Errorf("dag have no or multiple roots")
}
