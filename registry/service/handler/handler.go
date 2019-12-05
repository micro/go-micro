package handler

import (
	"context"
	"time"

	"github.com/micro/go-micro/errors"
	"github.com/micro/go-micro/registry"
	"github.com/micro/go-micro/registry/service"
	pb "github.com/micro/go-micro/registry/service/proto"
)

type Registry struct {
	// internal registry
	Registry registry.Registry
}

func (r *Registry) GetService(ctx context.Context, req *pb.GetRequest, rsp *pb.GetResponse) error {
	services, err := r.Registry.GetService(req.Service)
	if err != nil {
		return errors.InternalServerError("go.micro.registry", err.Error())
	}
	for _, srv := range services {
		rsp.Services = append(rsp.Services, service.ToProto(srv))
	}
	return nil
}

func (r *Registry) Register(ctx context.Context, req *pb.Service, rsp *pb.EmptyResponse) error {
	var regOpts []registry.RegisterOption
	if req.Options != nil {
		ttl := time.Duration(req.Options.Ttl) * time.Second
		regOpts = append(regOpts, registry.RegisterTTL(ttl))
	}

	err := r.Registry.Register(service.ToService(req), regOpts...)
	if err != nil {
		return errors.InternalServerError("go.micro.registry", err.Error())
	}

	return nil
}

func (r *Registry) Deregister(ctx context.Context, req *pb.Service, rsp *pb.EmptyResponse) error {
	err := r.Registry.Deregister(service.ToService(req))
	if err != nil {
		return errors.InternalServerError("go.micro.registry", err.Error())
	}
	return nil
}

func (r *Registry) ListServices(ctx context.Context, req *pb.ListRequest, rsp *pb.ListResponse) error {
	services, err := r.Registry.ListServices()
	if err != nil {
		return errors.InternalServerError("go.micro.registry", err.Error())
	}
	for _, srv := range services {
		rsp.Services = append(rsp.Services, service.ToProto(srv))
	}
	return nil
}

func (r *Registry) Watch(ctx context.Context, req *pb.WatchRequest, rsp pb.Registry_WatchStream) error {
	watcher, err := r.Registry.Watch(registry.WatchService(req.Service))
	if err != nil {
		return errors.InternalServerError("go.micro.registry", err.Error())
	}

	for {
		next, err := watcher.Next()
		if err != nil {
			return errors.InternalServerError("go.micro.registry", err.Error())
		}
		err = rsp.Send(&pb.Result{
			Action:  next.Action,
			Service: service.ToProto(next.Service),
		})
		if err != nil {
			return errors.InternalServerError("go.micro.registry", err.Error())
		}
	}
}
