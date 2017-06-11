package client

import (
	"bytes"
	"fmt"
	"sync"
	"time"

	"github.com/micro/go-micro/broker"
	"github.com/micro/go-micro/codec"
	"github.com/micro/go-micro/errors"
	"github.com/micro/go-micro/metadata"
	"github.com/micro/go-micro/selector"
	"github.com/micro/go-micro/transport"

	"golang.org/x/net/context"
)

type rpcClient struct {
	once sync.Once
	opts Options
	pool *pool
}

func newRpcClient(opt ...Option) Client {
	opts := newOptions(opt...)

	rc := &rpcClient{
		once: sync.Once{},
		opts: opts,
		pool: newPool(opts.PoolSize, opts.PoolTTL),
	}

	c := Client(rc)

	// wrap in reverse
	for i := len(opts.Wrappers); i > 0; i-- {
		c = opts.Wrappers[i-1](c)
	}

	return c
}

func (r *rpcClient) newCodec(contentType string) (codec.NewCodec, error) {
	if c, ok := r.opts.Codecs[contentType]; ok {
		return c, nil
	}
	if cf, ok := defaultCodecs[contentType]; ok {
		return cf, nil
	}
	return nil, fmt.Errorf("Unsupported Content-Type: %s", contentType)
}

func (r *rpcClient) call(ctx context.Context, address string, req Request, resp interface{}, opts CallOptions) error {
	msg := &transport.Message{
		Header: make(map[string]string),
	}

	md, ok := metadata.FromContext(ctx)
	if ok {
		for k, v := range md {
			msg.Header[k] = v
		}
	}

	// set timeout in nanoseconds
	msg.Header["Timeout"] = fmt.Sprintf("%d", opts.RequestTimeout)
	// set the content type for the request
	msg.Header["Content-Type"] = req.ContentType()
	// set the accept header
	msg.Header["Accept"] = req.ContentType()

	cf, err := r.newCodec(req.ContentType())
	if err != nil {
		return errors.InternalServerError("go.micro.client", err.Error())
	}

	var grr error
	c, err := r.pool.getConn(address, r.opts.Transport, transport.WithTimeout(opts.DialTimeout))
	if err != nil {
		return errors.InternalServerError("go.micro.client", "connection error: %v", err)
	}
	defer func() {
		// defer execution of release
		r.pool.release(address, c, grr)
	}()

	stream := &rpcStream{
		context: ctx,
		request: req,
		closed:  make(chan bool),
		codec:   newRpcPlusCodec(msg, c, cf),
	}
	defer stream.Close()

	ch := make(chan error, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				ch <- errors.InternalServerError("go.micro.client", "panic recovered: %v", r)
			}
		}()

		// send request
		if err := stream.Send(req.Request()); err != nil {
			ch <- err
			return
		}

		// recv request
		if err := stream.Recv(resp); err != nil {
			ch <- err
			return
		}

		// success
		ch <- nil
	}()

	select {
	case err := <-ch:
		grr = err
		return err
	case <-ctx.Done():
		grr = ctx.Err()
		return errors.New("go.micro.client", fmt.Sprintf("request timeout: %v", ctx.Err()), 408)
	}
}

func (r *rpcClient) stream(ctx context.Context, address string, req Request, opts CallOptions) (Streamer, error) {
	msg := &transport.Message{
		Header: make(map[string]string),
	}

	md, ok := metadata.FromContext(ctx)
	if ok {
		for k, v := range md {
			msg.Header[k] = v
		}
	}

	// set timeout in nanoseconds
	msg.Header["Timeout"] = fmt.Sprintf("%d", opts.RequestTimeout)
	// set the content type for the request
	msg.Header["Content-Type"] = req.ContentType()
	// set the accept header
	msg.Header["Accept"] = req.ContentType()

	cf, err := r.newCodec(req.ContentType())
	if err != nil {
		return nil, errors.InternalServerError("go.micro.client", err.Error())
	}

	c, err := r.opts.Transport.Dial(address, transport.WithStream(), transport.WithTimeout(opts.DialTimeout))
	if err != nil {
		return nil, errors.InternalServerError("go.micro.client", "connection error: %v", err)
	}

	stream := &rpcStream{
		context: ctx,
		request: req,
		closed:  make(chan bool),
		codec:   newRpcPlusCodec(msg, c, cf),
	}

	ch := make(chan error, 1)

	go func() {
		ch <- stream.Send(req.Request())
	}()

	var grr error

	select {
	case err := <-ch:
		grr = err
	case <-ctx.Done():
		grr = errors.New("go.micro.client", fmt.Sprintf("request timeout: %v", ctx.Err()), 408)
	}

	if grr != nil {
		stream.Close()
		return nil, grr
	}

	return stream, nil
}

func (r *rpcClient) Init(opts ...Option) error {
	size := r.opts.PoolSize
	ttl := r.opts.PoolTTL

	for _, o := range opts {
		o(&r.opts)
	}

	// recreate the pool if the options changed
	if size != r.opts.PoolSize || ttl != r.opts.PoolTTL {
		r.pool = newPool(r.opts.PoolSize, r.opts.PoolTTL)
	}

	return nil
}

func (r *rpcClient) Options() Options {
	return r.opts
}

func (r *rpcClient) CallRemote(ctx context.Context, address string, request Request, response interface{}, opts ...CallOption) error {
	// make a copy of call opts
	callOpts := r.opts.CallOptions
	for _, opt := range opts {
		opt(&callOpts)
	}
	return r.call(ctx, address, request, response, callOpts)
}

func (r *rpcClient) Call(ctx context.Context, request Request, response interface{}, opts ...CallOption) error {
	// make a copy of call opts
	callOpts := r.opts.CallOptions
	for _, opt := range opts {
		opt(&callOpts)
	}

	// get next nodes from the selector
	next, err := r.opts.Selector.Select(request.Service(), callOpts.SelectOptions...)
	if err != nil && err == selector.ErrNotFound {
		return errors.NotFound("go.micro.client", err.Error())
	} else if err != nil {
		return errors.InternalServerError("go.micro.client", err.Error())
	}

	// check if we already have a deadline
	d, ok := ctx.Deadline()
	if !ok {
		// no deadline so we create a new one
		ctx, _ = context.WithTimeout(ctx, callOpts.RequestTimeout)
	} else {
		// got a deadline so no need to setup context
		// but we need to set the timeout we pass along
		opt := WithRequestTimeout(d.Sub(time.Now()))
		opt(&callOpts)
	}

	// should we noop right here?
	select {
	case <-ctx.Done():
		return errors.New("go.micro.client", fmt.Sprintf("%v", ctx.Err()), 408)
	default:
	}

	// make copy of call method
	rcall := r.call

	// wrap the call in reverse
	for i := len(callOpts.CallWrappers); i > 0; i-- {
		rcall = callOpts.CallWrappers[i-1](rcall)
	}

	// return errors.New("go.micro.client", "request timeout", 408)
	call := func(i int) error {
		// call backoff first. Someone may want an initial start delay
		t, err := callOpts.Backoff(ctx, request, i)
		if err != nil {
			return errors.InternalServerError("go.micro.client", err.Error())
		}

		// only sleep if greater than 0
		if t.Seconds() > 0 {
			time.Sleep(t)
		}

		// select next node
		node, err := next()
		if err != nil && err == selector.ErrNotFound {
			return errors.NotFound("go.micro.client", err.Error())
		} else if err != nil {
			return errors.InternalServerError("go.micro.client", err.Error())
		}

		// set the address
		address := node.Address
		if node.Port > 0 {
			address = fmt.Sprintf("%s:%d", address, node.Port)
		}

		// make the call
		err = rcall(ctx, address, request, response, callOpts)
		r.opts.Selector.Mark(request.Service(), node, err)
		return err
	}

	ch := make(chan error, callOpts.Retries)
	var gerr error

	for i := 0; i < callOpts.Retries; i++ {
		go func() {
			ch <- call(i)
		}()

		select {
		case <-ctx.Done():
			return errors.New("go.micro.client", fmt.Sprintf("call timeout: %v", ctx.Err()), 408)
		case err := <-ch:
			// if the call succeeded lets bail early
			if err == nil {
				return nil
			}

			retry, rerr := callOpts.Retry(ctx, request, i, err)
			if rerr != nil {
				return rerr
			}

			if !retry {
				return err
			}

			gerr = err
		}
	}

	return gerr
}

func (r *rpcClient) StreamRemote(ctx context.Context, address string, request Request, opts ...CallOption) (Streamer, error) {
	// make a copy of call opts
	callOpts := r.opts.CallOptions
	for _, opt := range opts {
		opt(&callOpts)
	}
	return r.stream(ctx, address, request, callOpts)
}

func (r *rpcClient) Stream(ctx context.Context, request Request, opts ...CallOption) (Streamer, error) {
	// make a copy of call opts
	callOpts := r.opts.CallOptions
	for _, opt := range opts {
		opt(&callOpts)
	}

	// get next nodes from the selector
	next, err := r.opts.Selector.Select(request.Service(), callOpts.SelectOptions...)
	if err != nil && err == selector.ErrNotFound {
		return nil, errors.NotFound("go.micro.client", err.Error())
	} else if err != nil {
		return nil, errors.InternalServerError("go.micro.client", err.Error())
	}

	// check if we already have a deadline
	d, ok := ctx.Deadline()
	if !ok {
		// no deadline so we create a new one
		ctx, _ = context.WithTimeout(ctx, callOpts.RequestTimeout)
	} else {
		// got a deadline so no need to setup context
		// but we need to set the timeout we pass along
		opt := WithRequestTimeout(d.Sub(time.Now()))
		opt(&callOpts)
	}

	// should we noop right here?
	select {
	case <-ctx.Done():
		return nil, errors.New("go.micro.client", fmt.Sprintf("%v", ctx.Err()), 408)
	default:
	}

	call := func(i int) (Streamer, error) {
		// call backoff first. Someone may want an initial start delay
		t, err := callOpts.Backoff(ctx, request, i)
		if err != nil {
			return nil, errors.InternalServerError("go.micro.client", err.Error())
		}

		// only sleep if greater than 0
		if t.Seconds() > 0 {
			time.Sleep(t)
		}

		node, err := next()
		if err != nil && err == selector.ErrNotFound {
			return nil, errors.NotFound("go.micro.client", err.Error())
		} else if err != nil {
			return nil, errors.InternalServerError("go.micro.client", err.Error())
		}

		address := node.Address
		if node.Port > 0 {
			address = fmt.Sprintf("%s:%d", address, node.Port)
		}

		stream, err := r.stream(ctx, address, request, callOpts)
		r.opts.Selector.Mark(request.Service(), node, err)
		return stream, err
	}

	type response struct {
		stream Streamer
		err    error
	}

	ch := make(chan response, callOpts.Retries)
	var grr error

	for i := 0; i < callOpts.Retries; i++ {
		go func() {
			s, err := call(i)
			ch <- response{s, err}
		}()

		select {
		case <-ctx.Done():
			return nil, errors.New("go.micro.client", fmt.Sprintf("call timeout: %v", ctx.Err()), 408)
		case rsp := <-ch:
			// if the call succeeded lets bail early
			if rsp.err == nil {
				return rsp.stream, nil
			}

			retry, rerr := callOpts.Retry(ctx, request, i, rsp.err)
			if rerr != nil {
				return nil, rerr
			}

			if !retry {
				return nil, rsp.err
			}

			grr = rsp.err
		}
	}

	return nil, grr
}

func (r *rpcClient) Publish(ctx context.Context, p Publication, opts ...PublishOption) error {
	md, ok := metadata.FromContext(ctx)
	if !ok {
		md = make(map[string]string)
	}
	md["Content-Type"] = p.ContentType()

	// encode message body
	cf, err := r.newCodec(p.ContentType())
	if err != nil {
		return errors.InternalServerError("go.micro.client", err.Error())
	}
	b := &buffer{bytes.NewBuffer(nil)}
	if err := cf(b).Write(&codec.Message{Type: codec.Publication}, p.Message()); err != nil {
		return errors.InternalServerError("go.micro.client", err.Error())
	}
	r.once.Do(func() {
		r.opts.Broker.Connect()
	})

	return r.opts.Broker.Publish(p.Topic(), &broker.Message{
		Header: md,
		Body:   b.Bytes(),
	})
}

func (r *rpcClient) NewPublication(topic string, message interface{}) Publication {
	return newRpcPublication(topic, message, r.opts.ContentType)
}

func (r *rpcClient) NewProtoPublication(topic string, message interface{}) Publication {
	return newRpcPublication(topic, message, "application/octet-stream")
}
func (r *rpcClient) NewRequest(service, method string, request interface{}, reqOpts ...RequestOption) Request {
	return newRpcRequest(service, method, request, r.opts.ContentType, reqOpts...)
}

func (r *rpcClient) NewProtoRequest(service, method string, request interface{}, reqOpts ...RequestOption) Request {
	return newRpcRequest(service, method, request, "application/octet-stream", reqOpts...)
}

func (r *rpcClient) NewJsonRequest(service, method string, request interface{}, reqOpts ...RequestOption) Request {
	return newRpcRequest(service, method, request, "application/json", reqOpts...)
}

func (r *rpcClient) String() string {
	return "rpc"
}
