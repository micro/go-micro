package client

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/micro/go-micro/broker"
	"github.com/micro/go-micro/codec"
	"github.com/micro/go-micro/errors"
	"github.com/micro/go-micro/metadata"
	"github.com/micro/go-micro/registry"
	"github.com/micro/go-micro/selector"
	"github.com/micro/go-micro/transport"
)

type rpcClient struct {
	once sync.Once
	opts Options
	pool *pool
	seq  uint64
}

func newRpcClient(opt ...Option) Client {
	opts := newOptions(opt...)

	rc := &rpcClient{
		once: sync.Once{},
		opts: opts,
		pool: newPool(opts.PoolSize, opts.PoolTTL),
		seq:  0,
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
	if cf, ok := DefaultCodecs[contentType]; ok {
		return cf, nil
	}
	return nil, fmt.Errorf("Unsupported Content-Type: %s", contentType)
}

func (r *rpcClient) call(ctx context.Context, node *registry.Node, req Request, resp interface{}, opts CallOptions) error {
	address := node.Address
	if node.Port > 0 {
		address = fmt.Sprintf("%s:%d", address, node.Port)
	}

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

	// setup old protocol
	cf := setupProtocol(msg, node)

	// no codec specified
	if cf == nil {
		var err error
		cf, err = r.newCodec(req.ContentType())
		if err != nil {
			return errors.InternalServerError("go.micro.client", err.Error())
		}
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

	seq := atomic.LoadUint64(&r.seq)
	atomic.AddUint64(&r.seq, 1)
	codec := newRpcCodec(msg, c, cf)

	rsp := &rpcResponse{
		socket: c,
		codec:  codec,
	}

	stream := &rpcStream{
		context:  ctx,
		request:  req,
		response: rsp,
		codec:    codec,
		closed:   make(chan bool),
		id:       fmt.Sprintf("%v", seq),
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
		if err := stream.Send(req.Body()); err != nil {
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
		return errors.Timeout("go.micro.client", fmt.Sprintf("%v", ctx.Err()))
	}
}

func (r *rpcClient) stream(ctx context.Context, node *registry.Node, req Request, opts CallOptions) (Stream, error) {
	address := node.Address
	if node.Port > 0 {
		address = fmt.Sprintf("%s:%d", address, node.Port)
	}

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

	// set old codecs
	cf := setupProtocol(msg, node)

	// no codec specified
	if cf == nil {
		var err error
		cf, err = r.newCodec(req.ContentType())
		if err != nil {
			return nil, errors.InternalServerError("go.micro.client", err.Error())
		}
	}

	dOpts := []transport.DialOption{
		transport.WithStream(),
	}

	if opts.DialTimeout >= 0 {
		dOpts = append(dOpts, transport.WithTimeout(opts.DialTimeout))
	}

	c, err := r.opts.Transport.Dial(address, dOpts...)
	if err != nil {
		return nil, errors.InternalServerError("go.micro.client", "connection error: %v", err)
	}

	codec := newRpcCodec(msg, c, cf)

	rsp := &rpcResponse{
		socket: c,
		codec:  codec,
	}

	// set request codec
	if r, ok := req.(*rpcRequest); ok {
		r.codec = codec
	}

	stream := &rpcStream{
		context:  ctx,
		request:  req,
		response: rsp,
		closed:   make(chan bool),
		codec:    codec,
	}

	ch := make(chan error, 1)

	go func() {
		ch <- stream.Send(req.Body())
	}()

	var grr error

	select {
	case err := <-ch:
		grr = err
	case <-ctx.Done():
		grr = errors.Timeout("go.micro.client", fmt.Sprintf("%v", ctx.Err()))
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

	// update pool configuration if the options changed
	if size != r.opts.PoolSize || ttl != r.opts.PoolTTL {
		r.pool.Lock()
		r.pool.size = r.opts.PoolSize
		r.pool.ttl = int64(r.opts.PoolTTL.Seconds())
		r.pool.Unlock()
	}

	return nil
}

func (r *rpcClient) Options() Options {
	return r.opts
}

func (r *rpcClient) next(request Request, opts CallOptions) (selector.Next, error) {
	service := request.Service()

	// get proxy
	if prx := os.Getenv("MICRO_PROXY"); len(prx) > 0 {
		service = prx
	}

	// get proxy address
	if prx := os.Getenv("MICRO_PROXY_ADDRESS"); len(prx) > 0 {
		opts.Address = prx
	}

	// return remote address
	if len(opts.Address) > 0 {
		address := opts.Address
		port := 0

		host, sport, err := net.SplitHostPort(opts.Address)
		if err == nil {
			address = host
			port, _ = strconv.Atoi(sport)
		}

		return func() (*registry.Node, error) {
			return &registry.Node{
				Address: address,
				Port:    port,
				// Set the protocol
				Metadata: map[string]string{
					"protocol": "mucp",
				},
			}, nil
		}, nil
	}

	// get next nodes from the selector
	next, err := r.opts.Selector.Select(service, opts.SelectOptions...)
	if err != nil && err == selector.ErrNotFound {
		return nil, errors.NotFound("go.micro.client", "service %s: %v", service, err.Error())
	} else if err != nil {
		return nil, errors.InternalServerError("go.micro.client", "error selecting %s node: %v", service, err.Error())
	}

	return next, nil
}

func (r *rpcClient) Call(ctx context.Context, request Request, response interface{}, opts ...CallOption) error {
	// make a copy of call opts
	callOpts := r.opts.CallOptions
	for _, opt := range opts {
		opt(&callOpts)
	}

	next, err := r.next(request, callOpts)
	if err != nil {
		return err
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
		return errors.Timeout("go.micro.client", fmt.Sprintf("%v", ctx.Err()))
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
			return errors.InternalServerError("go.micro.client", "backoff error: %v", err.Error())
		}

		// only sleep if greater than 0
		if t.Seconds() > 0 {
			time.Sleep(t)
		}

		// select next node
		node, err := next()
		if err != nil && err == selector.ErrNotFound {
			return errors.NotFound("go.micro.client", "service %s: %v", request.Service(), err.Error())
		} else if err != nil {
			return errors.InternalServerError("go.micro.client", "error getting next %s node: %v", request.Service(), err.Error())
		}

		// make the call
		err = rcall(ctx, node, request, response, callOpts)
		r.opts.Selector.Mark(request.Service(), node, err)
		return err
	}

	ch := make(chan error, callOpts.Retries+1)
	var gerr error

	for i := 0; i <= callOpts.Retries; i++ {
		go func(i int) {
			ch <- call(i)
		}(i)

		select {
		case <-ctx.Done():
			return errors.Timeout("go.micro.client", fmt.Sprintf("call timeout: %v", ctx.Err()))
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

func (r *rpcClient) Stream(ctx context.Context, request Request, opts ...CallOption) (Stream, error) {
	// make a copy of call opts
	callOpts := r.opts.CallOptions
	for _, opt := range opts {
		opt(&callOpts)
	}

	next, err := r.next(request, callOpts)
	if err != nil {
		return nil, err
	}

	// should we noop right here?
	select {
	case <-ctx.Done():
		return nil, errors.Timeout("go.micro.client", fmt.Sprintf("%v", ctx.Err()))
	default:
	}

	call := func(i int) (Stream, error) {
		// call backoff first. Someone may want an initial start delay
		t, err := callOpts.Backoff(ctx, request, i)
		if err != nil {
			return nil, errors.InternalServerError("go.micro.client", "backoff error: %v", err.Error())
		}

		// only sleep if greater than 0
		if t.Seconds() > 0 {
			time.Sleep(t)
		}

		node, err := next()
		if err != nil && err == selector.ErrNotFound {
			return nil, errors.NotFound("go.micro.client", "service %s: %v", request.Service(), err.Error())
		} else if err != nil {
			return nil, errors.InternalServerError("go.micro.client", "error getting next %s node: %v", request.Service(), err.Error())
		}

		stream, err := r.stream(ctx, node, request, callOpts)
		r.opts.Selector.Mark(request.Service(), node, err)
		return stream, err
	}

	type response struct {
		stream Stream
		err    error
	}

	ch := make(chan response, callOpts.Retries+1)
	var grr error

	for i := 0; i <= callOpts.Retries; i++ {
		go func(i int) {
			s, err := call(i)
			ch <- response{s, err}
		}(i)

		select {
		case <-ctx.Done():
			return nil, errors.Timeout("go.micro.client", fmt.Sprintf("call timeout: %v", ctx.Err()))
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

func (r *rpcClient) Publish(ctx context.Context, msg Message, opts ...PublishOption) error {
	options := PublishOptions{
		Context: context.Background(),
	}
	for _, o := range opts {
		o(&options)
	}

	md, ok := metadata.FromContext(ctx)
	if !ok {
		md = make(map[string]string)
	}

	id := uuid.New().String()
	md["Content-Type"] = msg.ContentType()
	md["Micro-Topic"] = msg.Topic()
	md["Micro-Id"] = id

	// set the topic
	topic := msg.Topic()

	// get proxy
	if prx := os.Getenv("MICRO_PROXY"); len(prx) > 0 {
		options.Exchange = prx
	}

	// get the exchange
	if len(options.Exchange) > 0 {
		topic = options.Exchange
	}

	// encode message body
	cf, err := r.newCodec(msg.ContentType())
	if err != nil {
		return errors.InternalServerError("go.micro.client", err.Error())
	}
	b := &buffer{bytes.NewBuffer(nil)}
	if err := cf(b).Write(&codec.Message{
		Target: topic,
		Type:   codec.Publication,
		Header: map[string]string{
			"Micro-Id":    id,
			"Micro-Topic": msg.Topic(),
		},
	}, msg.Payload()); err != nil {
		return errors.InternalServerError("go.micro.client", err.Error())
	}
	r.once.Do(func() {
		r.opts.Broker.Connect()
	})

	return r.opts.Broker.Publish(topic, &broker.Message{
		Header: md,
		Body:   b.Bytes(),
	})
}

func (r *rpcClient) NewMessage(topic string, message interface{}, opts ...MessageOption) Message {
	return newMessage(topic, message, r.opts.ContentType, opts...)
}

func (r *rpcClient) NewRequest(service, method string, request interface{}, reqOpts ...RequestOption) Request {
	return newRequest(service, method, request, r.opts.ContentType, reqOpts...)
}

func (r *rpcClient) String() string {
	return "rpc"
}
