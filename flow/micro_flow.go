package flow

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/google/uuid"
	lru "github.com/hashicorp/golang-lru"
	"github.com/micro/go-micro/v2/errors"
	pb "github.com/micro/go-micro/v2/flow/service/proto"
	"github.com/micro/go-micro/v2/logger"
	"github.com/micro/go-micro/v2/metadata"
	"github.com/micro/go-micro/v2/store"
	"github.com/panjf2000/ants/v2"
)

type Status int

const (
	StatusUnknown Status = iota
	StatusPending
	StatusFailure
	StatusSuccess
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

func (fl *microFlow) readFlow(name string) ([]*pb.Step, error) {
	records, err := fl.options.FlowStore.Read(name)
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

	if err = fl.options.FlowStore.Write(&store.Record{Key: name, Value: buf}); err != nil {
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

// Result of the step flow
func (fl *microFlow) Result(flow string, rid string, step *Step) ([]byte, error) {
	return nil, nil
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

func (fl *microFlow) Status(flow string, rid string) (Status, error) {
	state := StatusUnknown
	records, err := fl.options.StateStore.Read(fmt.Sprintf("%s-%s-%s", flow, rid, "status"))
	if err != nil {
		return state, fmt.Errorf("flow status error %v flow %s rid %s", err, flow, rid)
	}

	switch string(records[0].Value) {
	case "pending":
		state = StatusPending
	case "failure":
		state = StatusFailure
	case "success":
		state = StatusSuccess
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
	err := fl.options.StateStore.Write(&store.Record{
		Key:   fmt.Sprintf("%s-%s-%s", flow, rid, "status"),
		Value: []byte("aborted"),
	})
	if err != nil {
		return fmt.Errorf("flow abort error %v flow %s rid %s", err, flow, rid)
	}
	return nil
}

func (fl *microFlow) Pause(flow string, rid string) error {
	err := fl.options.StateStore.Write(
		&store.Record{
			Key:   fmt.Sprintf("%s-%s-%s", flow, rid, "status"),
			Value: []byte("suspend"),
		})
	if err != nil {
		return fmt.Errorf("flow pause error %v flow %s rid %s", err, flow, rid)
	}
	return nil
}

func (fl *microFlow) Resume(flow string, rid string) error {
	err := fl.options.StateStore.Write(
		&store.Record{
			Key:   fmt.Sprintf("%s-%s-%s", flow, rid, "status"),
			Value: []byte("running"),
		})
	if err != nil {
		return fmt.Errorf("flow resume error %v flow %s rid %s", err, flow, rid)
	}
	return nil
}

func (fl *microFlow) Execute(flow string, req interface{}, rsp interface{}, opts ...ExecuteOption) (string, error) {
	var err error

	if !fl.initialized {
		return "", fmt.Errorf("initialize flow first")
	}

	options := ExecuteOptions{Flow: flow}
	for _, opt := range opts {
		opt(&options)
	}

	if options.Concurrency < 1 {
		options.Concurrency = DefaultExecuteConcurrency
	}

	if len(options.ID) == 0 {
		uid, err := uuid.NewRandom()
		if err != nil {
			return "", err
		}
		options.ID = uid.String()
		opts = append(opts, ExecuteID(options.ID))
	}

	var reqbuf []byte
	switch v := req.(type) {
	case *any.Any:
		reqbuf = v.Value
	case proto.Message:
		if reqbuf, err = proto.Marshal(v); err != nil {
			return "", err
		}
	case []byte:
		reqbuf = v
	default:
		return "", fmt.Errorf("req invalid, flow only works with proto.Message and []byte")
	}

	switch rsp.(type) {
	case *any.Any, proto.Message, []byte:
		break
	default:
		return "", fmt.Errorf("rsp invalid, flow only works with proto.Message and []byte")
	}

	job := &flowJob{flow: flow, req: reqbuf, options: opts, rid: options.ID}
	if !options.Async {
		job.done = make(chan struct{})
	}

	err = fl.pool.Invoke(job)
	if err != nil {
		return "", err
	}

	if !options.Async {
		job.mu.RLock()
		done := job.done
		job.mu.RUnlock()
		<-done
		job.mu.RLock()
		defer job.mu.RUnlock()
		if job.err != nil {
			return "", job.err
		}
		if job.rsp != nil {
			switch v := rsp.(type) {
			case *any.Any:
				v.Value = job.rsp
			case proto.Message:
				if err = proto.Unmarshal(job.rsp, v); err != nil {
					return "", err
				}
			case []byte:
				v = job.rsp
			}
		}
	}

	return options.ID, nil
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
		ants.WithPanicHandler(fl.options.ErrorHandler),
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
		_ = cached
		//modtime = fl.options.FlowStore.Modified(ctx, flow)
		//if modtime <= cached.timestamp {
		//	return cached.dag, nil
		//}
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
	fl.cache.Add(flow, &cacheDag{dag: g, timestamp: modtime})

	return g, nil
}

func (fl *microFlow) flowHandler(req interface{}) {
	var err error
	var serr error

	job := req.(*flowJob)
	defer func() {
		job.mu.Lock()
		(*job).err = err
		job.mu.Unlock()
	}()

	job.mu.RLock()
	if job.done != nil {
		defer close(job.done)
	}
	job.mu.RUnlock()

	options := ExecuteOptions{}
	for _, opt := range job.options {
		opt(&options)
	}

	if options.Context == nil {
		options.Context = fl.options.Context
	} else {
		options.Context = context.WithValue(options.Context, flowKey{}, fl)
	}

	var g dag
	g, err = fl.loadDag(options.Context, job.flow)
	if err != nil {
		return
	}

	var root *Step
	if len(options.Step) > 0 {
		if root, err = g.GetVertex(options.Step); err != nil {
			return
		}
	} else {
		root, err = g.GetRoot()
		if err != nil {
			return
		}
	}

	var steps []*Step

	if len(root.After) > 0 {
		if steps, err = g.OrderedAncestors(root); err != nil {
			return
		}
	}

	//	for _, s := range steps {
	//		fmt.Printf("TTT %v\n", s)
	//	}

	if steps, err = g.OrderedDescendants(root); err != nil {
		return
	}

	stateKey := fmt.Sprintf("%s-%s-flow", job.flow, job.rid)
	if serr = fl.options.StateStore.Write(&store.Record{Key: stateKey, Value: []byte("pending")}); serr != nil {
		err = fmt.Errorf("flow store key %s error %v", stateKey, serr)
		return
	}
	if logger.V(logger.TraceLevel, logger.DefaultLogger) {
		logger.Tracef("state %s %s", stateKey, "pending")
	}
	dataKey := fmt.Sprintf("%s-%s-%s", job.flow, job.rid, steps[0].Name())
	if serr := fl.options.DataStore.Write(&store.Record{Key: dataKey, Value: job.req}); serr != nil {
		err = fmt.Errorf("flow store key %s error %v", dataKey, serr)
		return
	}
	if logger.V(logger.TraceLevel, logger.DefaultLogger) {
		logger.Tracef("data %s %s", dataKey, job.req)
	}
	steps[0].Input = steps[0].Name()

stepsLoop:
	for idx, step := range steps {
		if len(step.Input) == 0 {
			step.Input = steps[idx-1].Output
		}
		if len(step.Output) == 0 {
			step.Output = step.Name()
		}
		if err = fl.stepHandler(options.Context, step, job); err != nil {
			break stepsLoop
		}
	}

	if err != nil {
		if serr := fl.options.StateStore.Write(&store.Record{Key: stateKey, Value: []byte("failure")}); serr != nil {
			err = fmt.Errorf("flow store key %s error %v", stateKey, serr)
			return
		}
		if logger.V(logger.TraceLevel, logger.DefaultLogger) {
			logger.Tracef("state %s %s", stateKey, "failure")
		}
		return
	}

	if serr = fl.options.StateStore.Write(&store.Record{Key: stateKey, Value: []byte("success")}); serr != nil {
		err = fmt.Errorf("flow store key %s error %v", stateKey, serr)
		return
	}
	if logger.V(logger.TraceLevel, logger.DefaultLogger) {
		logger.Tracef("state %s %s", stateKey, "success")
	}

	output := steps[0].Output
	if len(options.Output) > 0 {
		output = options.Output
	}

	dataKey = fmt.Sprintf("%s-%s-%s", job.flow, job.rid, output)
	rec, serr := fl.options.DataStore.Read(dataKey)
	if serr != nil {
		err = fmt.Errorf("flow store key %s error %v", dataKey, serr)
		return
	} else {
		job.rsp = rec[0].Value
	}
	if logger.V(logger.TraceLevel, logger.DefaultLogger) {
		logger.Tracef("data %s %s", dataKey, job.rsp)
	}

	return
}

func (fl *microFlow) stepHandler(ctx context.Context, step *Step, job *flowJob) error {
	var err error
	var serr error
	var buf []byte

	stateKey := fmt.Sprintf("%s-%s-%s", job.flow, job.rid, step.Name())
	if serr = fl.options.StateStore.Write(&store.Record{Key: stateKey, Value: []byte("pending")}); serr != nil {
		err = fmt.Errorf("flow store key %s error %v", stateKey, serr)
		return err
	}
	if logger.V(logger.TraceLevel, logger.DefaultLogger) {
		logger.Tracef("state %s %s", stateKey, "pending")
	}
	var rec []*store.Record
	var req []byte
	dataKey := fmt.Sprintf("%s-%s-%s", job.flow, job.rid, step.Input)
	rec, serr = fl.options.DataStore.Read(dataKey)
	if serr != nil {
		err = fmt.Errorf("flow store key %s error %v", dataKey, serr)
		return err
	}
	req = rec[0].Value
	if logger.V(logger.TraceLevel, logger.DefaultLogger) {
		logger.Tracef("data %s %s", dataKey, req)
	}

	dataKey = fmt.Sprintf("%s-%s-%s", job.flow, job.rid, step.Output)

	md, ok := metadata.FromContext(ctx)
	if !ok {
		md = make(metadata.Metadata)
	}

	var fallback bool
	var flowErr error

	// operation handles retries, timeouts and so
	buf, flowErr = step.Operation.Execute(metadata.NewContext(ctx, md), req, job.options...)
	if flowErr == nil {
		if serr := fl.options.StateStore.Write(&store.Record{Key: stateKey, Value: []byte("success")}); serr != nil {
			err = fmt.Errorf("flow store key %s error %v", stateKey, serr)
			return err
		}
		if logger.V(logger.TraceLevel, logger.DefaultLogger) {
			logger.Tracef("state %s %s", stateKey, "success")
		}
	} else {
		if step.Fallback != nil {
			fallback = true
		}

		if serr = fl.options.StateStore.Write(&store.Record{Key: stateKey, Value: []byte("failure")}); serr != nil {
			err = fmt.Errorf("flow store key %s error %v", stateKey, serr)
			return err
		}
		if logger.V(logger.TraceLevel, logger.DefaultLogger) {
			logger.Tracef("state %s %s", stateKey, "failure")
		}
		if m, ok := flowErr.(*errors.Error); ok {
			buf, serr = proto.Marshal(m)
			if serr != nil {
				return serr
			}
		} else {
			buf = []byte(flowErr.Error())
		}
	}

	if serr = fl.options.DataStore.Write(&store.Record{Key: dataKey, Value: buf}); serr != nil {
		err = fmt.Errorf("flow store key %s error %v", dataKey, serr)
		return err
	}
	if logger.V(logger.TraceLevel, logger.DefaultLogger) {
		logger.Tracef("data %s %s", dataKey, buf)
	}

	if fallback {
		if logger.V(logger.TraceLevel, logger.DefaultLogger) {
			logger.Tracef("fallback operation provided")
		}
		buf, err = step.Fallback.Execute(metadata.NewContext(ctx, md), req, job.options...)
		if err == nil {
			if serr := fl.options.StateStore.Write(&store.Record{Key: stateKey, Value: []byte("success")}); serr != nil {
				err = fmt.Errorf("flow store key %s error %v", stateKey, serr)
				return err
			}
			if logger.V(logger.TraceLevel, logger.DefaultLogger) {
				logger.Tracef("state %s %s", stateKey, "success")
			}
		} else {
			if serr = fl.options.StateStore.Write(&store.Record{Key: stateKey, Value: []byte("failure")}); serr != nil {
				err = fmt.Errorf("flow store key %s error %v", stateKey, serr)
				return err
			}
			if logger.V(logger.TraceLevel, logger.DefaultLogger) {
				logger.Tracef("state %s %s", stateKey, "failure")
			}
			if m, ok := err.(*errors.Error); ok {
				buf, serr = proto.Marshal(m)
				if serr != nil {
					return serr
				}
			} else {
				buf = []byte(err.Error())
			}
		}

		if serr = fl.options.DataStore.Write(&store.Record{Key: dataKey, Value: buf}); serr != nil {
			err = fmt.Errorf("flow store key %s error %v", dataKey, serr)
			return err
		}
		if logger.V(logger.TraceLevel, logger.DefaultLogger) {
			logger.Tracef("data %s %s", dataKey, buf)
		}
	}

	return flowErr
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
