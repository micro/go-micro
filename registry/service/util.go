package service

import (
	"github.com/micro/go-micro/v2/registry"
	pb "github.com/micro/go-micro/v2/registry/service/proto"
)

func values(v []*registry.Value) []*pb.Value {
	if len(v) == 0 {
		return []*pb.Value{}
	}

	vs := make([]*pb.Value, 0, len(v))
	for _, vi := range v {
		vs = append(vs, &pb.Value{
			Name:   vi.Name,
			Type:   vi.Type,
			Values: values(vi.Values),
		})
	}
	return vs
}

func toValues(v []*pb.Value) []*registry.Value {
	if len(v) == 0 {
		return []*registry.Value{}
	}

	vs := make([]*registry.Value, 0, len(v))
	for _, vi := range v {
		vs = append(vs, &registry.Value{
			Name:   vi.Name,
			Type:   vi.Type,
			Values: toValues(vi.Values),
		})
	}
	return vs
}

func ToProto(s *registry.Service) *pb.Service {
	endpoints := make([]*pb.Endpoint, 0, len(s.Endpoints))
	for _, ep := range s.Endpoints {
		var request, response *pb.Value

		if ep.Request != nil {
			request = &pb.Value{
				Name:   ep.Request.Name,
				Type:   ep.Request.Type,
				Values: values(ep.Request.Values),
			}
		}

		if ep.Response != nil {
			response = &pb.Value{
				Name:   ep.Response.Name,
				Type:   ep.Response.Type,
				Values: values(ep.Response.Values),
			}
		}

		endpoints = append(endpoints, &pb.Endpoint{
			Name:     ep.Name,
			Request:  request,
			Response: response,
			Metadata: ep.Metadata,
		})
	}

	nodes := make([]*pb.Node, 0, len(s.Nodes))

	for _, node := range s.Nodes {
		nodes = append(nodes, &pb.Node{
			Id:       node.Id,
			Address:  node.Address,
			Metadata: node.Metadata,
		})
	}

	return &pb.Service{
		Name:      s.Name,
		Version:   s.Version,
		Metadata:  s.Metadata,
		Endpoints: endpoints,
		Nodes:     nodes,
		Options:   new(pb.Options),
	}
}

func ToService(s *pb.Service) *registry.Service {
	endpoints := make([]*registry.Endpoint, 0, len(s.Endpoints))
	for _, ep := range s.Endpoints {
		var request, response *registry.Value

		if ep.Request != nil {
			request = &registry.Value{
				Name:   ep.Request.Name,
				Type:   ep.Request.Type,
				Values: toValues(ep.Request.Values),
			}
		}

		if ep.Response != nil {
			response = &registry.Value{
				Name:   ep.Response.Name,
				Type:   ep.Response.Type,
				Values: toValues(ep.Response.Values),
			}
		}

		endpoints = append(endpoints, &registry.Endpoint{
			Name:     ep.Name,
			Request:  request,
			Response: response,
			Metadata: ep.Metadata,
		})
	}

	nodes := make([]*registry.Node, 0, len(s.Nodes))
	for _, node := range s.Nodes {
		nodes = append(nodes, &registry.Node{
			Id:       node.Id,
			Address:  node.Address,
			Metadata: node.Metadata,
		})
	}

	return &registry.Service{
		Name:      s.Name,
		Version:   s.Version,
		Metadata:  s.Metadata,
		Endpoints: endpoints,
		Nodes:     nodes,
	}
}
