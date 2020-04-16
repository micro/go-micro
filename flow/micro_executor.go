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
	"github.com/micro/go-micro/v2/logger"
	"github.com/micro/go-micro/v2/metadata"
	"github.com/micro/go-micro/v2/store"
	"github.com/panjf2000/ants/v2"
)

type cacheDag struct {
	dag       dag
	timestamp int64
}

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
	steps   []*Step
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

type microExecutor struct {
	dataStore  store.Store
	stateStore store.Store
	options    ExecutorOptions
	pool       *ants.PoolWithFunc
	cache      *lru.TwoQueueCache

	workers  int
	wait     bool
	nonblock bool
	prealloc bool
	sync.RWMutex
	initialized bool
}

func newMicroExecutor() Executor {
	fl := &microExecutor{}
	fl.options.Context = context.Background()
	return fl
}

func (fl *microExecutor) Init(opts ...ExecutorOption) error {
	for _, o := range opts {
		o(&fl.options)
	}

	if s, ok := fl.options.Context.Value(stateStoreOptionKey{}).(store.Store); ok && s != nil {
		fl.stateStore = s
	}
	if s, ok := fl.options.Context.Value(dataStoreOptionKey{}).(store.Store); ok && s != nil {
		fl.dataStore = s
	}

	cache, err := lru.New2Q(100)
	if err != nil {
		return err
	}

	if fl.workers < 1 {
		fl.workers = DefaultConcurrency
	}
	pool, err := ants.NewPoolWithFunc(
		fl.workers,
		fl.flowHandler,
		ants.WithNonblocking(fl.nonblock),
		ants.WithPanicHandler(fl.options.ErrorHandler),
		ants.WithPreAlloc(fl.prealloc),
	)
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

func (fl *microExecutor) Options() ExecutorOptions {
	return fl.options
}

// Result of the step flow
func (fl *microExecutor) Result(flow string, rid string, step string) ([]byte, error) {
	return nil, nil
}

func (fl *microExecutor) Status(flow string, rid string) (Status, error) {
	state := StatusUnknown
	records, err := fl.stateStore.Read(fmt.Sprintf("%s-%s-%s", flow, rid, "status"))
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

func (fl *microExecutor) Abort(flow string, rid string) error {
	err := fl.stateStore.Write(&store.Record{
		Key:   fmt.Sprintf("%s-%s-%s", flow, rid, "status"),
		Value: []byte("aborted"),
	})
	if err != nil {
		return fmt.Errorf("flow abort error %v flow %s rid %s", err, flow, rid)
	}
	return nil
}

func (fl *microExecutor) Pause(flow string, rid string) error {
	err := fl.stateStore.Write(
		&store.Record{
			Key:   fmt.Sprintf("%s-%s-%s", flow, rid, "status"),
			Value: []byte("suspend"),
		})
	if err != nil {
		return fmt.Errorf("flow pause error %v flow %s rid %s", err, flow, rid)
	}
	return nil
}

func (fl *microExecutor) Resume(flow string, rid string) error {
	err := fl.stateStore.Write(
		&store.Record{
			Key:   fmt.Sprintf("%s-%s-%s", flow, rid, "status"),
			Value: []byte("running"),
		})
	if err != nil {
		return fmt.Errorf("flow resume error %v flow %s rid %s", err, flow, rid)
	}
	return nil
}

func (fl *microExecutor) Execute(steps []*Step, req interface{}, rsp interface{}, opts ...ExecuteOption) (string, error) {
	var err error

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

	job := &flowJob{steps: steps, flow: options.Flow, req: reqbuf, options: opts, rid: options.ID}
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

func (fl *microExecutor) flowHandler(req interface{}) {
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

	steps := job.steps

	options := ExecuteOptions{}
	for _, opt := range job.options {
		opt(&options)
	}

	if options.Context == nil {
		options.Context = fl.options.Context
	} else {
		options.Context = context.WithValue(options.Context, flowKey{}, fl)
	}

	stateKey := fmt.Sprintf("%s-%s-flow", job.flow, job.rid)
	if serr = fl.stateStore.Write(&store.Record{Key: stateKey, Value: []byte("pending")}); serr != nil {
		err = fmt.Errorf("flow store key %s error %v", stateKey, serr)
		return
	}
	if logger.V(logger.TraceLevel, logger.DefaultLogger) {
		logger.Tracef("state %s %s", stateKey, "pending")
	}
	dataKey := fmt.Sprintf("%s-%s-%s", job.flow, job.rid, steps[0].Name())
	if serr := fl.dataStore.Write(&store.Record{Key: dataKey, Value: job.req}); serr != nil {
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
		if serr := fl.stateStore.Write(&store.Record{Key: stateKey, Value: []byte("failure")}); serr != nil {
			err = fmt.Errorf("flow store key %s error %v", stateKey, serr)
			return
		}
		if logger.V(logger.TraceLevel, logger.DefaultLogger) {
			logger.Tracef("state %s %s", stateKey, "failure")
		}
		return
	}

	if serr = fl.stateStore.Write(&store.Record{Key: stateKey, Value: []byte("success")}); serr != nil {
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
	rec, serr := fl.dataStore.Read(dataKey)
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

func (fl *microExecutor) stepHandler(ctx context.Context, step *Step, job *flowJob) error {
	var err error
	var serr error
	var buf []byte

	stateKey := fmt.Sprintf("%s-%s-%s", job.flow, job.rid, step.Name())
	if serr = fl.stateStore.Write(&store.Record{Key: stateKey, Value: []byte("pending")}); serr != nil {
		err = fmt.Errorf("flow store key %s error %v", stateKey, serr)
		return err
	}
	if logger.V(logger.TraceLevel, logger.DefaultLogger) {
		logger.Tracef("state %s %s", stateKey, "pending")
	}
	var rec []*store.Record
	var req []byte
	dataKey := fmt.Sprintf("%s-%s-%s", job.flow, job.rid, step.Input)
	rec, serr = fl.dataStore.Read(dataKey)
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
		if serr := fl.stateStore.Write(&store.Record{Key: stateKey, Value: []byte("success")}); serr != nil {
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

		if serr = fl.stateStore.Write(&store.Record{Key: stateKey, Value: []byte("failure")}); serr != nil {
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

	if serr = fl.dataStore.Write(&store.Record{Key: dataKey, Value: buf}); serr != nil {
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
			if serr := fl.stateStore.Write(&store.Record{Key: stateKey, Value: []byte("success")}); serr != nil {
				err = fmt.Errorf("flow store key %s error %v", stateKey, serr)
				return err
			}
			if logger.V(logger.TraceLevel, logger.DefaultLogger) {
				logger.Tracef("state %s %s", stateKey, "success")
			}
		} else {
			if serr = fl.stateStore.Write(&store.Record{Key: stateKey, Value: []byte("failure")}); serr != nil {
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

		if serr = fl.dataStore.Write(&store.Record{Key: dataKey, Value: buf}); serr != nil {
			err = fmt.Errorf("flow store key %s error %v", dataKey, serr)
			return err
		}
		if logger.V(logger.TraceLevel, logger.DefaultLogger) {
			logger.Tracef("data %s %s", dataKey, buf)
		}
	}

	return flowErr
}

func (fl *microExecutor) Stop() error {
	fl.Lock()
	fl.pool.Release()
	fl.Unlock()
	if fl.wait {
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
