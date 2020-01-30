// Package service uses the flow service
package service

import (
	"context"

	"github.com/micro/go-micro/v2/client"
	"github.com/micro/go-micro/v2/flow"
	pb "github.com/micro/go-micro/v2/flow/service/proto"
)

var (
	// The default service name
	DefaultService = "go.micro.flow"
)

type serviceFlow struct {
	name string
	opts flow.Options
	// client to call flow service
	client pb.FlowService
}

func (s *serviceFlow) Init(opts ...flow.Option) error {
	for _, o := range opts {
		o(&s.opts)
	}
	return nil
}

func (s *serviceFlow) Options() flow.Options {
	return s.opts
}

func (s *serviceFlow) CreateStep(fname string, step *flow.Step) error {
	pbStep := stepToProto(step)

	// create step
	_, err := s.client.CreateStep(context.TODO(), &pb.CreateStepRequest{Flow: fname, Step: pbStep})
	if err != nil {
		return err
	}

	return nil
}

func (s *serviceFlow) Pause(fname string, reqID string) error {
	return nil
}
func (s *serviceFlow) Resume(fname string, reqID string) error {
	return nil
}
func (s *serviceFlow) Abort(fname string, reqID string) error {
	return nil
}
func (s *serviceFlow) Stop() error {
	return nil
}

func (s *serviceFlow) Execute(fname string, req interface{}, rsp interface{}, opts ...flow.ExecuteOption) (string, error) {
	return "", nil
}

func (s *serviceFlow) ReplaceStep(fname string, oldstep *flow.Step, newstep *flow.Step) error {
	return nil
}

func (s *serviceFlow) DeleteStep(fname string, step *flow.Step) error {
	pbStep := stepToProto(step)

	// delete step
	_, err := s.client.DeleteStep(context.TODO(), &pb.DeleteStepRequest{Flow: fname, Step: pbStep})
	if err != nil {
		return err
	}
	return nil
}

func (s *serviceFlow) String() string {
	return s.name
}

// NewFlow returns a new flow service client
func NewFlow(opts ...flow.Option) flow.Flow {
	var options flow.Options
	for _, o := range opts {
		o(&options)
	}

	// service name
	// TODO: accept option
	name := DefaultService

	return &serviceFlow{
		opts:   options,
		name:   name,
		client: pb.NewFlowService(name, client.DefaultClient),
	}
}
