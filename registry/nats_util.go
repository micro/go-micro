//go:build nats
// +build nats

package registry

func cp(current []*Service) []*Service {
	var services []*Service

	for _, service := range current {
		// copy service
		s := new(Service)
		*s = *service

		// copy nodes
		var nodes []*Node
		for _, node := range service.Nodes {
			n := new(Node)
			*n = *node
			nodes = append(nodes, n)
		}
		s.Nodes = nodes

		// copy endpoints
		var eps []*Endpoint
		for _, ep := range service.Endpoints {
			e := new(Endpoint)
			*e = *ep
			eps = append(eps, e)
		}
		s.Endpoints = eps

		// append service
		services = append(services, s)
	}

	return services
}

func addNodes(old, neu []*Node) []*Node {
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

func addServices(old, neu []*Service) []*Service {
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

func delNodes(old, del []*Node) []*Node {
	var nodes []*Node
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

func delServices(old, del []*Service) []*Service {
	var services []*Service
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
