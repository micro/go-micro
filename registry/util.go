package registry

func addNodes(old, neu []*Node) []*Node {
	nodes := make([]*Node, len(neu))
	// add all new nodes
	for i, n := range neu {
		node := *n
		nodes[i] = &node
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

// CopyService make a copy of service
func CopyService(service *Service) *Service {
	// copy service
	s := new(Service)
	*s = *service

	// copy nodes
	nodes := make([]*Node, len(service.Nodes))
	for j, node := range service.Nodes {
		n := new(Node)
		*n = *node
		nodes[j] = n
	}
	s.Nodes = nodes

	// copy endpoints
	eps := make([]*Endpoint, len(service.Endpoints))
	for j, ep := range service.Endpoints {
		e := new(Endpoint)
		*e = *ep
		eps[j] = e
	}
	s.Endpoints = eps
	return s
}

// Copy makes a copy of services
func Copy(current []*Service) []*Service {
	services := make([]*Service, len(current))
	for i, service := range current {
		services[i] = CopyService(service)
	}
	return services
}

// Merge merges two lists of services and returns a new copy
func Merge(olist []*Service, nlist []*Service) []*Service {
	var srv []*Service

	for _, n := range nlist {
		var seen bool
		for _, o := range olist {
			if o.Version == n.Version {
				sp := new(Service)
				// make copy
				*sp = *o
				// set nodes
				sp.Nodes = addNodes(o.Nodes, n.Nodes)

				// mark as seen
				seen = true
				srv = append(srv, sp)
				break
			} else {
				sp := new(Service)
				// make copy
				*sp = *o
				srv = append(srv, sp)
			}
		}
		if !seen {
			srv = append(srv, Copy([]*Service{n})...)
		}
	}
	return srv
}

// Remove removes services and returns a new copy
func Remove(old, del []*Service) []*Service {
	var services []*Service

	for _, o := range old {
		srv := new(Service)
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
