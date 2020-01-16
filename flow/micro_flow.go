package flow

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	pbFlow "github.com/micro/go-micro/flow/proto"
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

func (fl *microFlow) CreateStep(ctx context.Context, name string, step *Step) error {
	var err error
	var buf []byte

	steps := &pbFlow.Steps{}

	buf, err = fl.options.FlowStore.Read(ctx, name)
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
	log.Printf("%#+v\n", steps.Steps)
	if buf, err = proto.Marshal(steps); err != nil {
		return err
	}

	if err = fl.options.FlowStore.Write(ctx, name, buf); err != nil {
		return err
	}

	return nil
}

func (fl *microFlow) DeleteStep(ctx context.Context, name string, step *Step) error {
	return nil
}

func (fl *microFlow) Abort(ctx context.Context, name string, reqID string) error {
	return nil
}

func (fl *microFlow) Pause(ctx context.Context, flow string, rid string) error {
	return nil
}

func (fl *microFlow) Resume(ctx context.Context, flow string, rid string) error {
	return nil
}

func (fl *microFlow) Execute(ctx context.Context, flow string, req interface{}, rsp interface{}, opts ...ExecuteOption) (string, error) {

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

	job := &flowJob{flow: flow, req: reqbuf, options: options}
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
		fl.handler,
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

func (fl *microFlow) handler(r interface{}) {
	req := r.(*flowJob)
	if req.done != nil {
		defer close(req.done)
	}

	buf, err := fl.options.FlowStore.Read(req.options.Context, req.flow)
	if err != nil {
		//	panic(err)
		(*req).err = err
		return
	}
	_ = buf
	log.Printf("%#+v\n", req)
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
