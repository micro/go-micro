// +build ignore

package flow

import (
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	dag "github.com/hashicorp/terraform/dag"
	pbFlow "github.com/micro/go-micro/flow/service/proto"
	"github.com/panjf2000/ants/v2"
)

type walkerStep struct {
	step *Step
	pos  int
}

type walker struct {
	steps []walkerStep
}

func (w *walker) Walk(n dag.Vertex, pos int) error {
	w.steps = append(w.steps, walkerStep{step: n.(*Step), pos: pos})
	return nil
}

type flowJob struct {
	flow    string
	step    string
	req     []byte
	rsp     []byte
	err     error
	done    chan struct{}
	options []ExecuteOption
}

var (
	DefaultConcurrency        = 100
	DefaultExecuteConcurrency = 10
)

type microFlow struct {
	sync.RWMutex
	options Options
	pool    *ants.PoolWithFunc
}

// Create default executor
func NewFlow(opts ...Option) Flow {
	options := Options{}
	for _, opt := range opts {
		opt(&options)
	}

	fl := &microFlow{
		options: options,
	}

	return fl
}

func (fl *microFlow) ReplaceStep(name string, oldstep *Step, newstep *Step) error {
	var err error
	var buf []byte

	if buf, err = fl.options.FlowStore.Read(fl.options.Context, name); err != nil {
		return err
	}

	steps := &pbFlow.Steps{}
	if err = proto.Unmarshal(buf, steps); err != nil {
		return err
	}

	st := stepToProto(oldstep)
	for idx, pbst := range steps.Steps {
		if stepEqual(pbst, st) {
			steps.Steps = append(steps.Steps[:idx], steps.Steps[idx+1:]...)
		}
	}

	steps.Steps = append(steps.Steps, stepToProto(newstep))
	if buf, err = proto.Marshal(steps); err != nil {
		return err
	}

	if err = fl.options.FlowStore.Write(fl.options.Context, name, buf); err != nil {
		return err
	}

	return nil
}

func (fl *microFlow) CreateStep(name string, step *Step) error {
	var err error
	var buf []byte

	steps := &pbFlow.Steps{}

	buf, err = fl.options.FlowStore.Read(fl.options.Context, name)
	switch err {
	case nil:
		if err := proto.Unmarshal(buf, steps); err != nil {
			return err
		}
	case ErrFlowNotFound:
		steps.Steps = make([]*pbFlow.Step, 0, 0)
		break
	default:
		return err
	}

	steps.Steps = append(steps.Steps, stepToProto(step))
	if buf, err = proto.Marshal(steps); err != nil {
		return err
	}

	if err = fl.options.FlowStore.Write(fl.options.Context, name, buf); err != nil {
		return err
	}

	return nil
}

func (fl *microFlow) Lookup(name string) ([]*Step, error) {
	var err error
	var buf []byte

	if buf, err = fl.options.FlowStore.Read(fl.options.Context, name); err != nil {
		return nil, err
	}

	pbSteps := &pbFlow.Steps{}
	if err = proto.Unmarshal(buf, pbSteps); err != nil {
		return nil, err
	}

	steps := make([]*Step, 0, len(pbSteps.Steps))
	for _, step := range pbSteps.Steps {
		steps = append(steps, protoToStep(step))
	}

	return steps, nil
}

func (fl *microFlow) DeleteStep(name string, step *Step) error {
	var err error
	var buf []byte

	if buf, err = fl.options.FlowStore.Read(fl.options.Context, name); err != nil {
		return err
	}

	steps := &pbFlow.Steps{}
	if err = proto.Unmarshal(buf, steps); err != nil {
		return err
	}

	st := stepToProto(step)

	for idx, pbst := range steps.Steps {
		if stepEqual(pbst, st) {
			steps.Steps = append(steps.Steps[:idx], steps.Steps[idx+1:]...)
		}
	}

	if buf, err = proto.Marshal(steps); err != nil {
		return err
	}

	if err = fl.options.FlowStore.Write(fl.options.Context, name, buf); err != nil {
		return err
	}

	return nil
}

func (fl *microFlow) Abort(name string, reqID string) error {
	return nil
}

func (fl *microFlow) Pause(flow string, rid string) error {
	return nil
}

func (fl *microFlow) Resume(flow string, rid string) error {
	return nil
}

func (fl *microFlow) Execute(flow string, step string, req interface{}, rsp interface{}, opts ...ExecuteOption) (string, error) {

	if fl.pool == nil {
		return "", fmt.Errorf("initialize flow first")
	}

	options := ExecuteOptions{}
	for _, opt := range opts {
		opt(&options)
	}
	if options.Concurrency < 1 {
		options.Concurrency = DefaultExecuteConcurrency
	}

	uid, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}

	reqmsg, ok := req.(proto.Message)
	if !ok {
		return "", fmt.Errorf("req invalid, flow only works with proto.Message now")
	}

	rspmsg, ok := rsp.(proto.Message)
	if !ok {
		return "", fmt.Errorf("rsp invalid, flow only works with proto.Message now")
	}

	reqbuf, err := proto.Marshal(reqmsg)
	if err != nil {
		return "", err
	}

	job := &flowJob{flow: flow, step: step, req: reqbuf, options: opts}
	if !options.Async {
		job.done = make(chan struct{})
	}

	err = fl.pool.Invoke(job)
	if err != nil {
		return "", err
	}

	if !options.Async {
		<-job.done
		if job.err != nil {
			return "", job.err
		}
		if job.rsp != nil {
			if err = proto.Unmarshal(job.rsp, rspmsg); err != nil {
				return "", err
			}
		}
	}

	return uid.String(), nil
}

func (fl *microFlow) Init(opts ...Option) error {
	for _, opt := range opts {
		opt(&fl.options)
	}

	if fl.options.Concurrency < 1 {
		fl.options.Concurrency = DefaultConcurrency
	}
	pool, err := ants.NewPoolWithFunc(
		fl.options.Concurrency,
		fl.flowHandler,
		ants.WithNonblocking(fl.options.Nonblock),
		ants.WithPanicHandler(fl.options.PanicHandler),
		ants.WithPreAlloc(fl.options.Prealloc),
	)
	if err != nil {
		return err
	}

	fl.Lock()
	fl.pool = pool
	fl.Unlock()

	return nil
}

func include(slice []string, f string) bool {
	for _, s := range slice {
		if s == f {
			return true
		}
	}
	return false
}

func (fl *microFlow) flowHandler(req interface{}) {
	var err error
	var buf []byte

	job := req.(*flowJob)
	defer func() {
		(*job).err = err
	}()

	if job.done != nil {
		defer close(job.done)
	}

	options := ExecuteOptions{}
	for _, opt := range job.options {
		opt(&options)
	}

	if buf, err = fl.options.FlowStore.Read(options.Context, job.flow); err != nil {
		return
	}

	pbSteps := &pbFlow.Steps{}
	if err = proto.Unmarshal(buf, pbSteps); err != nil {
		return
	}

	steps := make(map[string]*Step)
	for _, step := range pbSteps.Steps {
		s := protoToStep(step)
		steps[s.Name()] = s
	}

	g := &dag.AcyclicGraph{}
	for _, s := range steps {
		g.Add(s)
	}

	for _, vs := range steps {
	requiresLoop:
		for _, req := range vs.Requires {
			if req == "all" {
				for _, ve := range steps {
					if ve.Name() != vs.Name() && !include(ve.Requires, "all") {
						g.Connect(dag.BasicEdge(ve, vs))
					}
				}
				break requiresLoop
			}
			ve, ok := steps[req]
			if !ok {
				err = fmt.Errorf("requires unknown step %v", vs)
				return
			}
			g.Connect(dag.BasicEdge(ve, vs))
		}
	requiredLoop:
		for _, req := range vs.Required {
			if req == "all" {
				for _, ve := range steps {
					if ve.Name() != vs.Name() && !include(ve.Required, "all") {
						g.Connect(dag.BasicEdge(vs, ve))
					}
				}
				break requiredLoop
			}
			ve, ok := steps[req]
			if !ok {
				err = fmt.Errorf("required unknown step %v", vs)
				return
			}
			g.Connect(dag.BasicEdge(vs, ve))
		}

	}
	if err = g.Validate(); err != nil {
		return
	}

	g.TransitiveReduction()

	var root dag.Vertex
	if root, err = g.Root(); err != nil {
		return
	}

	w := &walker{}
	err = g.DepthFirstWalk([]dag.Vertex{root}, w.Walk)
	if err != nil {
		return
	}

	// sort steps for forward execution
	sort.Slice(w.steps, func(i, j int) bool {
		return w.steps[i].pos < w.steps[j].pos
	})

	for _, wstep := range w.steps {
		log.Printf("step %s\n", wstep.step.Name())
		for _, op := range wstep.step.Operations {
			log.Printf("op %s\n", op.Name())
			buf, err = op.Execute(options.Context, job.req, job.options...)
			if err != nil {
				return
			}
		}
	}

}

func (fl *microFlow) Options() Options {
	return fl.options
}

func (fl *microFlow) Stop() error {
	fl.Lock()
	fl.pool.Release()
	fl.Unlock()
	if fl.options.Wait {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
	loop:
		for {
			select {
			case <-fl.options.Context.Done():
				break loop
			case <-ticker.C:
				if fl.pool.Running() == 0 {
					break loop
				}
			}
		}
	}

	return nil
}

/*
type FlowOperation struct {
	Node     string   `json:"node"`
	Service  string   `json:"service"`
	Endpoint string   `json:"endpoint"`
	Requires []string `json:"requires"`
	Required []string `json:"required"`

	Options   []client.CallOption `json:"options"`
	Aggregate bool                `json:"aggregate"`
}

func (f *flowManager) Init(opts ...ManagerOption) error {
	options := ManagerOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	f.opts = options
	/*
	   pool, err := ants.NewPoolWithFunc(
	     f.opts.Concurrency,
	     w.Handle,
	     ants.WithPreAlloc(true),
	   )
	   if err != nil {
	   return err
	   }
	   f.pool = pool
*/

/*
	return nil
}

func (f *flowManager) Options() ManagerOptions {
	return f.opts
}

func (f *flowManager) Subscribe(flow string, service *FlowOperation) error {
	// uuid1 AccountCreate TokenCreate
	// uuid2 AccountCreate ContactCreate
	// uuid3 AccountCreate NetworkCreate
	if err := f.opts.FlowStore.Append(flow, service); err != nil {
		return err
	}

	return nil
}

func (f *flowManager) Unsubscribe(flow string, service *FlowOperation) error {
	// uuid1 AccountCreate TokenCreate
	// uuid2 AccountCreate ContactCreate
	// uuid3 AccountCreate NetworkCreate
	if err := f.opts.FlowStore.Delete(flow, service); err != nil {
		return err
	}

	return nil
}

func (f *flowManager) Execute(flow string, req proto.Message, rsp proto.Message, opts ...ExecuteOption) (string, error) {
	reqbuf, err := proto.Marshal(req)
	if err != nil {
		return "", err
	}

	rspbuf, rid, err := f.opts.Executor.Execute(flow, reqbuf, opts...)
	if err != nil {
		return "", err
	}

	if rspbuf != nil {
		if err = proto.Unmarshal(rspbuf, rsp); err != nil {
			return "", err
		}
	}

	return rid, nil
}

func (f *flowManager) Pause(flow string, rid string) error {
	return f.opts.Executor.Pause(flow, rid)
}

func (f *flowManager) Resume(flow string, rid string) error {
	return f.opts.Executor.Resume(flow, rid)
}

func (f *flowManager) Stop(flow string, rid string) error {
	return f.opts.Executor.Stop(flow, rid)
}

func (f *flowManager) Lookup(flow string, rid string, rsp interface{}) error {
	return nil
}

func (f *flowManager) Export(flow string) ([]byte, error) {
	return nil, nil
}

func (f *flowManager) Import(flow string, data []byte) error {
	return nil
}
*/
