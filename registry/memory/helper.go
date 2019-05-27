package memory

import (
	"github.com/micro/go-micro/registry"
)

func addNodes(old, neu []*registry.Node) []*registry.Node {
	for _, n := range neu {
		var seen bool
		for i, o := range old {
			if o.Id == n.Id {
				seen = true
				old[i] = n
				break
			}
		}
		if !seen {
			old = append(old, n)
		}
	}
	return old
}

func addServices(old, neu []*registry.Service) []*registry.Service {
	for _, s := range neu {
		var seen bool
		for i, o := range old {
			if o.Version == s.Version {
				s.Nodes = addNodes(o.Nodes, s.Nodes)
				seen = true
				old[i] = s
				break
			}
		}
		if !seen {
			old = append(old, s)
		}
	}
	return old
}

func delNodes(old, del []*registry.Node) []*registry.Node {
	var nodes []*registry.Node
	for _, o := range old {
		var rem bool
		for _, n := range del {
			if o.Id == n.Id {
				rem = true
				break
			}
		}
		if !rem {
			nodes = append(nodes, o)
		}
	}
	return nodes
}

func delServices(old, del []*registry.Service) []*registry.Service {
	var services []*registry.Service
	for i, o := range old {
		var rem bool
		for _, s := range del {
			if o.Version == s.Version {
				old[i].Nodes = delNodes(o.Nodes, s.Nodes)
				if len(old[i].Nodes) == 0 {
					rem = true
				}
			}
		}
		if !rem {
			services = append(services, o)
		}
	}
	return services
}
