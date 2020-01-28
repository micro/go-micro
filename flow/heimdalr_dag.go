package flow

import (
	"fmt"

	hdag "github.com/heimdalr/dag"
)

type heimdalrDag struct {
	dag   *hdag.DAG
	names map[string]interface{}
}

func NewHeimdalrDag() *heimdalrDag {
	return &heimdalrDag{
		dag:   hdag.NewDAG(),
		names: make(map[string]interface{}),
	}
}

func (g *heimdalrDag) AddVertex(v interface{}) error {
	vn, ok := v.(hdag.Vertex)
	if !ok {
		return fmt.Errorf("vertex have no Name() method")
	}
	g.names[vn.Id()] = v
	return g.dag.AddVertex(vn)
}

func (g *heimdalrDag) AddEdge(src interface{}, dst interface{}) error {
	vsrc, ok := src.(hdag.Vertex)
	if !ok {
		return fmt.Errorf("vertex have no Name() method")
	}
	vdst, ok := dst.(hdag.Vertex)
	if !ok {
		return fmt.Errorf("vertex have no Name() method")
	}

	return g.dag.AddEdge(vsrc, vdst)
}

func (g *heimdalrDag) OrderedAncestors(v interface{}) ([]*Step, error) {
	vn, ok := v.(hdag.Vertex)
	if !ok {
		return nil, fmt.Errorf("vertex have no Name() method")
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
		return nil, fmt.Errorf("vertex have no Name() method")
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
	v, ok := g.names[name]
	if !ok {
		return nil, fmt.Errorf("vertex not found")
	}
	return v, nil
}
