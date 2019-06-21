package registry

import (
	"bytes"
	"compress/zlib"
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"strings"
)

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
	var nodes []*Node

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

func addServices(old, neu []*Service) []*Service {
	var srv []*Service

	for _, s := range neu {
		var seen bool
		for _, o := range old {
			if o.Version == s.Version {
				sp := new(Service)
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
			srv = append(srv, cp([]*Service{s})...)
		}
	}

	return srv
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

func encode(txt *mdnsTxt) ([]string, error) {
	b, err := json.Marshal(txt)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	defer buf.Reset()

	w := zlib.NewWriter(&buf)
	if _, err := w.Write(b); err != nil {
		return nil, err
	}
	w.Close()

	encoded := hex.EncodeToString(buf.Bytes())

	// individual txt limit
	if len(encoded) <= 255 {
		return []string{encoded}, nil
	}

	// split encoded string
	var record []string

	for len(encoded) > 255 {
		record = append(record, encoded[:255])
		encoded = encoded[255:]
	}

	record = append(record, encoded)

	return record, nil
}

func decode(record []string) (*mdnsTxt, error) {
	encoded := strings.Join(record, "")

	hr, err := hex.DecodeString(encoded)
	if err != nil {
		return nil, err
	}

	br := bytes.NewReader(hr)
	zr, err := zlib.NewReader(br)
	if err != nil {
		return nil, err
	}

	rbuf, err := ioutil.ReadAll(zr)
	if err != nil {
		return nil, err
	}

	var txt *mdnsTxt

	if err := json.Unmarshal(rbuf, &txt); err != nil {
		return nil, err
	}

	return txt, nil
}
