package handler

import (
	"context"

	"github.com/micro/go-micro/errors"
	"github.com/micro/go-micro/runtime"
	pb "github.com/micro/go-micro/runtime/service/proto"
)

type Runtime struct {
	Runtime runtime.Runtime
}

func toProto(s *runtime.Service) *pb.Service {
	return &pb.Service{
		Name:     s.Name,
		Version:  s.Version,
		Source:   s.Source,
		Metadata: s.Metadata,
	}
}

func toService(s *pb.Service) *runtime.Service {
	return &runtime.Service{
		Name:     s.Name,
		Version:  s.Version,
		Source:   s.Source,
		Metadata: s.Metadata,
	}
}

func toCreateOptions(opts *pb.CreateOptions) []runtime.CreateOption {
	options := []runtime.CreateOption{}
	// command options
	if len(opts.Command) > 0 {
		options = append(options, runtime.WithCommand(opts.Command...))
	}
	// env options
	if len(opts.Env) > 0 {
		options = append(options, runtime.WithEnv(opts.Env))
	}

	// TODO: output options

	return options
}

func toReadOptions(opts *pb.ReadOptions) []runtime.ReadOption {
	options := []runtime.ReadOption{}
	if len(opts.Service) > 0 {
		options = append(options, runtime.ReadService(opts.Service))
	}
	if len(opts.Version) > 0 {
		options = append(options, runtime.ReadVersion(opts.Version))
	}
	if len(opts.Type) > 0 {
		options = append(options, runtime.ReadType(opts.Type))
	}

	return options
}

func (r *Runtime) Create(ctx context.Context, req *pb.CreateRequest, rsp *pb.CreateResponse) error {
	if req.Service == nil {
		return errors.BadRequest("go.micro.runtime", "blank service")
	}

	var options []runtime.CreateOption
	if req.Options != nil {
		options = toCreateOptions(req.Options)
	}

	service := toService(req.Service)
	err := r.Runtime.Create(service, options...)
	if err != nil {
		return errors.InternalServerError("go.micro.runtime", err.Error())
	}

	return nil
}

func (r *Runtime) Read(ctx context.Context, req *pb.ReadRequest, rsp *pb.ReadResponse) error {
	var options []runtime.ReadOption
	if req.Options != nil {
		options = toReadOptions(req.Options)
	}

	services, err := r.Runtime.Read(options...)
	if err != nil {
		return errors.InternalServerError("go.micro.runtime", err.Error())
	}

	for _, service := range services {
		rsp.Services = append(rsp.Services, toProto(service))
	}

	return nil
}

func (r *Runtime) Update(ctx context.Context, req *pb.UpdateRequest, rsp *pb.UpdateResponse) error {
	if req.Service == nil {
		return errors.BadRequest("go.micro.runtime", "blank service")
	}

	// TODO: add opts
	service := toService(req.Service)
	err := r.Runtime.Update(service)
	if err != nil {
		return errors.InternalServerError("go.micro.runtime", err.Error())
	}

	return nil
}

func (r *Runtime) Delete(ctx context.Context, req *pb.DeleteRequest, rsp *pb.DeleteResponse) error {
	if req.Service == nil {
		return errors.BadRequest("go.micro.runtime", "blank service")
	}

	// TODO: add opts
	service := toService(req.Service)
	err := r.Runtime.Delete(service)
	if err != nil {
		return errors.InternalServerError("go.micro.runtime", err.Error())
	}

	return nil
}

func (r *Runtime) List(ctx context.Context, req *pb.ListRequest, rsp *pb.ListResponse) error {
	services, err := r.Runtime.List()
	if err != nil {
		return errors.InternalServerError("go.micro.runtime", err.Error())
	}

	for _, service := range services {
		rsp.Services = append(rsp.Services, toProto(service))
	}

	return nil
}
