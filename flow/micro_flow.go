package flow

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	lru "github.com/hashicorp/golang-lru"
	"github.com/panjf2000/ants/v2"
)

type Status int

const (
	StatusUnknown Status = iota
	StatusPaused
	StatusAborted
	StatusStopped
)

type flowJob struct {
	flow    string
	rid     string
	step    string
	req     []byte
	rsp     []byte
	err     error
	done    chan struct{}
	options []ExecuteOption
	wg      sync.WaitGroup
	mu      sync.RWMutex
}

var (
	DefaultConcurrency        = 100
	DefaultExecuteConcurrency = 10
)

type cacheDag struct {
	dag       dag
	timestamp int64
}

type microFlow struct {
	sync.RWMutex
	options     Options
	pool        *ants.PoolWithFunc
	cache       *lru.TwoQueueCache
	initialized bool
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

func (fl *microFlow) ReplaceStep(flow string, oldstep *Step, newstep *Step) error {
	steps, err := fl.options.FlowStore.Load(fl.options.Context, flow)
	if err != nil {
		return err
	}

	for idx, step := range steps {
		if stepEqual(step, oldstep) {
			steps[idx] = newstep
		}
	}

	if err = fl.options.FlowStore.Save(fl.options.Context, flow, steps); err != nil {
		return err
	}

	return nil
}

func (fl *microFlow) CreateStep(flow string, step *Step) error {
	steps, err := fl.options.FlowStore.Load(fl.options.Context, flow)
	if err != nil && err != ErrFlowNotFound {
		return err
	}

	steps = append(steps, step)

	if err = fl.options.FlowStore.Save(fl.options.Context, flow, steps); err != nil {
		return err
	}

	return nil
}

// Lookup flow
func (fl *microFlow) Lookup(flow string) ([]*Step, error) {
	steps, err := fl.options.FlowStore.Load(fl.options.Context, flow)
	if err != nil {
		return nil, err
	}

	return steps, nil
}

func (fl *microFlow) DeleteStep(flow string, step *Step) error {
	steps, err := fl.options.FlowStore.Load(fl.options.Context, flow)
	if err != nil {
		return err
	}

	for idx, curstep := range steps {
		if stepEqual(curstep, step) {
			steps = append(steps[:idx], steps[idx+1:]...)
		}
	}

	if err = fl.options.FlowStore.Save(fl.options.Context, flow, steps); err != nil {
		return err
	}

	return nil
}

func (fl *microFlow) Status(flow string, rid string) (Status, error) {
	state := StatusUnknown
	buf, err := fl.options.StateStore.Read(fl.options.Context, flow, rid, []byte("status"))
	if err != nil {
		return state, err
	}

	switch string(buf) {
	case "aborted":
		state = StatusAborted
	case "paused":
		state = StatusPaused
	case "stopped":
		state = StatusAborted
	}

	return state, nil
}

func (fl *microFlow) Abort(flow string, rid string) error {
	return fl.options.StateStore.Write(fl.options.Context, flow, rid, []byte("status"), []byte("aborted"))
}

func (fl *microFlow) Pause(flow string, rid string) error {
	return fl.options.StateStore.Write(fl.options.Context, flow, rid, []byte("status"), []byte("suspend"))
}

func (fl *microFlow) Resume(flow string, rid string) error {
	return fl.options.StateStore.Write(fl.options.Context, flow, rid, []byte("status"), []byte("running"))
}

func (fl *microFlow) Execute(flow string, step string, req interface{}, rsp interface{}, opts ...ExecuteOption) (string, error) {
	if !fl.initialized {
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

	var reqbuf []byte
	switch v := req.(type) {
	case proto.Message:
		reqbuf, err = proto.Marshal(v)
	case []byte:
		reqbuf = v
	default:
		return "", fmt.Errorf("req invalid, flow only works with proto.Message and []byte")
	}

	rspmsg, ok := rsp.(proto.Message)
	if !ok && rsp != nil {
		return "", fmt.Errorf("rsp invalid, flow only works with proto.Message and []byte")
	}

	job := &flowJob{flow: flow, step: step, req: reqbuf, options: opts, rid: uid.String()}
	if !options.Async {
		job.done = make(chan struct{})
	}

	err = fl.pool.Invoke(job)
	if err != nil {
		return "", err
	}

	job.mu.Lock()
	defer job.mu.Unlock()
	if !options.Async {
		<-job.done
		if job.err != nil {
			return "", job.err
		}
		if job.rsp != nil && rspmsg != nil {
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

	if fl.options.Context == nil {
		fl.options.Context = context.Background()
	}

	fl.options.Context = FlowToContext(fl.options.Context, fl)

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

	cache, err := lru.New2Q(100)
	if err != nil {
		return err
	}

	fl.Lock()
	fl.pool = pool
	fl.cache = cache
	fl.initialized = true
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

func (fl *microFlow) loadDag(ctx context.Context, flow string) (dag, error) {
	var modtime int64
	cdag, ok := fl.cache.Get(flow)
	if ok {
		cached := cdag.(*cacheDag)
		modtime = fl.options.FlowStore.Modified(ctx, flow)
		if modtime <= cached.timestamp {
			return cached.dag, nil
		}
	}

	steps, err := fl.options.FlowStore.Load(ctx, flow)
	if err != nil {
		return nil, err
	}

	g := NewHeimdalrDag()
	stepsMap := make(map[string]*Step)
	for _, s := range steps {
		stepsMap[s.Name()] = s
		g.AddVertex(s)
	}

	for _, vs := range steps {
	requiresLoop:
		for _, req := range vs.Requires {
			if req == "all" {
				for _, ve := range steps {
					if ve.Name() != vs.Name() && !include(ve.Requires, "all") {
						g.AddEdge(ve, vs)
					}
				}
				break requiresLoop
			}
			ve, ok := stepsMap[req]
			if !ok {
				err = fmt.Errorf("requires unknown step %v", vs)
				return nil, err
			}
			g.AddEdge(ve, vs)
		}
	requiredLoop:
		for _, req := range vs.Required {
			if req == "all" {
				for _, ve := range steps {
					if ve.Name() != vs.Name() && !include(ve.Required, "all") {
						g.AddEdge(vs, ve)
					}
				}
				break requiredLoop
			}
			ve, ok := stepsMap[req]
			if !ok {
				err = fmt.Errorf("required unknown step %v", vs)
				return nil, err
			}
			g.AddEdge(vs, ve)
		}

	}
	if err = g.Validate(); err != nil {
		return nil, err
	}

	g.TransitiveReduction()
	fl.cache.Add(flow, &cacheDag{dag: g, timestamp: modtime})

	return g, nil
}

func (fl *microFlow) flowHandler(req interface{}) {
	var err error

	job := req.(*flowJob)
	//job.mu.Lock()
	//defer job.mu.Unlock()
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

	if options.Context == nil {
		options.Context = fl.options.Context
	} else {
		options.Context = context.WithValue(options.Context, flowKey{}, fl)
	}

	g, err := fl.loadDag(options.Context, job.flow)
	if err != nil {
		return
	}

	var root interface{}
	if root, err = g.GetVertex(job.step); err != nil {
		return
	}

	var steps []*Step
	if steps, err = g.OrderedDescendants(root); err != nil {
		return
	}

	initial := true
stepsLoop:
	for _, step := range steps {
		if err = fl.stepHandler(options.Context, step, job, initial); err != nil {
			initial = false
			if step.Fallback != nil {
				fl.stepHandler(options.Context, step, job, initial)
			}
			break stepsLoop
		}
		if initial {
			initial = false
		}
	}
}

func (fl *microFlow) stepHandler(ctx context.Context, step *Step, job *flowJob, initial bool) error {
	var err error
	var opErr error
	var buf []byte

	stateName := fmt.Sprintf("%s-%s", step.Name(), step.Operation.Name())
	if err = fl.options.StateStore.Write(ctx, job.flow, job.rid, []byte(stateName), []byte("pending")); err != nil {
		return err
	}

	// operation handles retries, timeouts and so
	buf, opErr = step.Operation.Execute(ctx, job.req, job.options...)
	if opErr == nil {
		if err = fl.options.StateStore.Write(ctx, job.flow, job.rid, []byte(stateName), []byte("complete")); err != nil {
			return err
		}
		if err = fl.options.DataStore.Write(ctx, job.flow, job.rid, []byte(stateName), buf); err != nil {
			return err
		}
		if initial {
			job.rsp = buf
		}
		return nil
	}

	if err = fl.options.StateStore.Write(ctx, job.flow, job.rid, []byte(stateName), []byte("failure")); err != nil {
		return err
	}

	return opErr
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
