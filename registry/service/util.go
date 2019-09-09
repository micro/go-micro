package service

import (
	"github.com/micro/go-micro/registry"
	pb "github.com/micro/go-micro/registry/proto"
)

func values(v []*registry.Value) []*pb.Value {
	if len(v) == 0 {
		return []*pb.Value{}
	}

	var vs []*pb.Value
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

	var vs []*registry.Value
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
	var endpoints []*pb.Endpoint
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

	var nodes []*pb.Node

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
	}
}

func ToService(s *pb.Service) *registry.Service {
	var endpoints []*registry.Endpoint
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

	var nodes []*registry.Node
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
