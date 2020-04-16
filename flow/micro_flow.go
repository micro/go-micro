package flow

import (
	"context"
	"fmt"
	"sync"

	"github.com/golang/protobuf/proto"
	pb "github.com/micro/go-micro/v2/flow/service/proto"
	"github.com/micro/go-micro/v2/store"
)

type microFlow struct {
	sync.RWMutex
	options     Options
	initialized bool
	flowStore   store.Store
}

// Create default executor
func newMicroFlow(opts ...Option) Flow {
	fl := &microFlow{}
	fl.options.Context = context.Background()
	for _, opt := range opts {
		opt(&fl.options)
	}

	defstore := DefaultStore
	if s, ok := fl.options.Context.Value(storeOptionKey{}).(store.Store); ok && s != nil {
		defstore = s
	}
	fl.flowStore = defstore

	if s, ok := fl.options.Context.Value(flowStoreOptionKey{}).(store.Store); ok && s != nil {
		fl.flowStore = s
	}

	return fl
}

func (fl *microFlow) readFlow(name string) ([]*pb.Step, error) {
	records, err := fl.flowStore.Read(name)
	if err != nil {
		return nil, err
	}

	rec := records[0]

	pbsteps := &pb.Steps{}
	if err = proto.Unmarshal(rec.Value, pbsteps); err != nil {
		return nil, err
	}

	return pbsteps.Steps, nil
}

func (fl *microFlow) writeFlow(name string, pbsteps []*pb.Step) error {
	buf, err := proto.Marshal(&pb.Steps{Steps: pbsteps})
	if err != nil {
		return err
	}

	if err = fl.flowStore.Write(&store.Record{Key: name, Value: buf}); err != nil {
		return err
	}

	return nil
}

func (fl *microFlow) ReplaceStep(flow string, oldstep *Step, newstep *Step) error {
	fl.Lock()
	defer fl.Unlock()

	pbsteps, err := fl.readFlow(flow)
	if err != nil {
		return err
	}

	var found bool
	for idx, pbstep := range pbsteps {
		if pbstep.Name == oldstep.Name() {
			pbsteps[idx] = StepToProto(newstep)
			found = true
		}
	}

	if !found {
		return fmt.Errorf("step %s not found", oldstep.Name())
	}

	if err = fl.writeFlow(flow, pbsteps); err != nil {
		return err
	}

	return nil
}

func (fl *microFlow) CreateStep(flow string, step *Step) error {
	fl.Lock()
	defer fl.Unlock()

	pbsteps, err := fl.readFlow(flow)
	if err != nil && err != store.ErrNotFound {
		return err
	}

	for _, pbstep := range pbsteps {
		if pbstep.Name == step.Name() {
			return ErrStepExists
		}
	}

	pbsteps = append(pbsteps, StepToProto(step))

	if err = fl.writeFlow(flow, pbsteps); err != nil {
		return err
	}

	return nil
}

// Lookup flow steps
func (fl *microFlow) Lookup(flow string) ([]*Step, error) {
	fl.RLock()
	defer fl.RUnlock()

	pbsteps, err := fl.readFlow(flow)
	if err != nil {
		return nil, err
	}

	steps := make([]*Step, 0, len(pbsteps))
	for _, pbstep := range pbsteps {
		steps = append(steps, ProtoToStep(pbstep))
	}

	return steps, nil
}

func (fl *microFlow) DeleteStep(flow string, step *Step) error {
	fl.Lock()
	defer fl.Unlock()

	pbsteps, err := fl.readFlow(flow)
	if err != nil {
		return err
	}

	var found bool
	for idx, pbstep := range pbsteps {
		step := ProtoToStep(pbstep)
		if pbstep.Name == step.Name() {
			pbsteps = append(pbsteps[:idx], pbsteps[idx+1:]...)
			found = true
		}
	}

	if !found {
		return fmt.Errorf("step %s not found", step.Name())
	}

	if err = fl.writeFlow(flow, pbsteps); err != nil {
		return err
	}

	return nil
}

func (fl *microFlow) Init(opts ...Option) error {
	for _, opt := range opts {
		opt(&fl.options)
	}

	if fl.options.Context == nil {
		fl.options.Context = context.Background()
	}

	fl.options.Context = FlowToContext(fl.options.Context, fl)

	if fl.options.Executor == nil {
		fl.options.Executor = newMicroExecutor()
	}

	fl.options.Executor.Init(
		WithExecutorContext(fl.options.Context),
	)

	return nil
}

func (fl *microFlow) Options() Options {
	return fl.options
}

func (fl *microFlow) loadDag(flow string) (dag, error) {

	include := func(slice []string, f string) bool {
		for _, s := range slice {
			if s == f {
				return true
			}
		}
		return false
	}

	steps, err := fl.Lookup(flow)
	if err != nil {
		return nil, err
	}

	g := newHeimdalrDag()
	stepsMap := make(map[string]*Step)
	for _, s := range steps {
		stepsMap[s.Name()] = s
		g.AddVertex(s)
	}

	for _, vs := range steps {
	afterLoop:
		for _, req := range vs.After {
			if req == "all" {
				for _, ve := range steps {
					if ve.Name() != vs.Name() && !include(ve.After, "all") {
						g.AddEdge(ve, vs)
					}
				}
				break afterLoop
			}
			ve, ok := stepsMap[req]
			if !ok {
				err = fmt.Errorf("%v after unknown step %v", vs, req)
				return nil, err
			}
			g.AddEdge(ve, vs)
		}
	beforeLoop:
		for _, req := range vs.Before {
			if req == "all" {
				for _, ve := range steps {
					if ve.Name() != vs.Name() && !include(ve.Before, "all") {
						g.AddEdge(vs, ve)
					}
				}
				break beforeLoop
			}
			ve, ok := stepsMap[req]
			if !ok {
				err = fmt.Errorf("%v before unknown step %v", vs, req)
				return nil, err
			}
			g.AddEdge(vs, ve)
		}

	}
	if err = g.Validate(); err != nil {
		return nil, err
	}

	g.TransitiveReduction()

	return g, nil
}

func (fl *microFlow) Execute(req interface{}, rsp interface{}, opts ...ExecuteOption) (string, error) {
	var err error
	var g dag

	options := ExecuteOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	g, err = fl.loadDag(options.Flow)
	if err != nil {
		return "", err
	}

	var root *Step
	if len(options.Step) > 0 {
		if root, err = g.GetVertex(options.Step); err != nil {
			return "", err
		}
	} else {
		root, err = g.GetRoot()
		if err != nil {
			return "", err
		}
	}

	var steps []*Step

	if len(root.After) > 0 {
		if steps, err = g.OrderedAncestors(root); err != nil {
			return "", err
		}
	}

	if steps, err = g.OrderedDescendants(root); err != nil {
		return "", err
	}

	return fl.options.Executor.Execute(steps, req, rsp, opts...)
}
