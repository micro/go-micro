package gossip

import (
	"github.com/micro/go-micro/registry"
)

func cp(current []*registry.Service) []*registry.Service {
	var services []*registry.Service

	for _, service := range current {
		// copy service
		s := new(registry.Service)
		*s = *service

		// copy nodes
		var nodes []*registry.Node
		for _, node := range service.Nodes {
			n := new(registry.Node)
			*n = *node
			nodes = append(nodes, n)
		}
		s.Nodes = nodes

		// copy endpoints
		var eps []*registry.Endpoint
		for _, ep := range service.Endpoints {
			e := new(registry.Endpoint)
			*e = *ep
			eps = append(eps, e)
		}
		s.Endpoints = eps

		// append service
		services = append(services, s)
	}

	return services
}

func addNodes(old, neu []*registry.Node) []*registry.Node {
	var nodes []*registry.Node

	// add all new nodes
	for _, n := range neu {
		node := *n
		nodes = append(nodes, &node)
	}

	// look at old nodes
	for _, o := range old {
		var exists bool

		// check against new nodes
		for _, n := range nodes {
			// ids match then skip
			if o.Id == n.Id {
				exists = true
				break
			}
		}

		// keep old node
		if !exists {
			node := *o
			nodes = append(nodes, &node)
		}
	}

	return nodes
}

func addServices(old, neu []*registry.Service) []*registry.Service {
	var srv []*registry.Service

	for _, s := range neu {
		var seen bool
		for _, o := range old {
			if o.Version == s.Version {
				sp := new(registry.Service)
				// make copy
				*sp = *o
				// set nodes
				sp.Nodes = addNodes(o.Nodes, s.Nodes)

				// mark as seen
				seen = true
				srv = append(srv, sp)
				break
			}
		}
		if !seen {
			srv = append(srv, cp([]*registry.Service{s})...)
		}
	}

	return srv
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

	for _, o := range old {
		srv := new(registry.Service)
		*srv = *o

		var rem bool

		for _, s := range del {
			if srv.Version == s.Version {
				srv.Nodes = delNodes(srv.Nodes, s.Nodes)

				if len(srv.Nodes) == 0 {
					rem = true
				}
			}
		}

		if !rem {
			services = append(services, srv)
		}
	}

	return services
}
