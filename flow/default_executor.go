package flow

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/google/uuid"
	"github.com/panjf2000/ants/v2"
)

//go:generate protoc --go_out=paths=source_relative:. proto/message.proto

////go:generate protoc --go_out=paths=source_relative:. --micro_out=paths=source_relative:. proto/message.proto

type flowJob struct {
	flow    string
	req     []byte
	rsp     []byte
	err     error
	done    chan struct{}
	options ExecuteOptions
}

var (
	DefaultExecutorConcurrency = 100
	DefaultExecuteConcurrency  = 10
)

type defaultExecutor struct {
	sync.RWMutex
	options ExecutorOptions
	pool    *ants.PoolWithFunc
}

// Create default executor
func NewExecutor(opts ...ExecutorOption) Executor {
	options := ExecutorOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	exc := &defaultExecutor{
		options: options,
	}

	return exc
}

func (exc *defaultExecutor) FlowAbort(ctx context.Context, flow string, rid string) error {
	return nil
}

func (exc *defaultExecutor) FlowPause(ctx context.Context, flow string, rid string) error {
	return nil
}

func (exc *defaultExecutor) FlowResume(ctx context.Context, flow string, rid string) error {
	return nil
}

func (exc *defaultExecutor) FlowExecute(ctx context.Context, flow string, req interface{}, rsp interface{}, opts ...ExecuteOption) (string, error) {

	if exc.pool == nil {
		return "", fmt.Errorf("executor not initialized")
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

	job := &flowJob{flow: flow, req: reqbuf, options: options}
	if !options.Async {
		job.done = make(chan struct{})
	}

	err = exc.pool.Invoke(job)
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

func (exc *defaultExecutor) Init(opts ...ExecutorOption) error {
	for _, opt := range opts {
		opt(&exc.options)
	}

	if exc.options.Concurrency < 1 {
		exc.options.Concurrency = DefaultExecutorConcurrency
	}
	pool, err := ants.NewPoolWithFunc(
		exc.options.Concurrency,
		exc.handler,
		ants.WithNonblocking(exc.options.Nonblock),
		ants.WithPanicHandler(exc.options.PanicHandler),
		ants.WithPreAlloc(exc.options.Prealloc),
	)
	if err != nil {
		return err
	}

	exc.Lock()
	exc.pool = pool
	exc.Unlock()

	return nil
}

func (exc *defaultExecutor) handler(r interface{}) {
	req := r.(*flowJob)
	if req.done != nil {
		defer close(req.done)
	}

	buf, err := exc.options.FlowStore.Read(req.options.Context, req.flow)
	if err != nil {
		//	panic(err)
		(*req).err = err
		return
	}
	_ = buf
	log.Printf("%#+v\n", req)
}

func (exc *defaultExecutor) Options() ExecutorOptions {
	return exc.options
}

func (exc *defaultExecutor) Stop() error {
	exc.Lock()
	exc.pool.Release()
	exc.Unlock()
	if exc.options.Wait {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
	loop:
		for {
			select {
			case <-exc.options.Context.Done():
				break loop
			case <-ticker.C:
				if exc.pool.Running() == 0 {
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
