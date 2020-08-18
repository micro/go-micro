// Package mucp provides an mucp client
package mucp

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/micro/go-micro/v3/broker"
	"github.com/micro/go-micro/v3/client"
	"github.com/micro/go-micro/v3/codec"
	raw "github.com/micro/go-micro/v3/codec/bytes"
	"github.com/micro/go-micro/v3/errors"
	"github.com/micro/go-micro/v3/metadata"
	"github.com/micro/go-micro/v3/transport"
	"github.com/micro/go-micro/v3/util/buf"
	"github.com/micro/go-micro/v3/util/pool"
)

type rpcClient struct {
	once atomic.Value
	opts client.Options
	pool pool.Pool
	seq  uint64
}

// NewClient returns a new micro client interface
func NewClient(opt ...client.Option) client.Client {
	opts := client.NewOptions(opt...)

	p := pool.NewPool(
		pool.Size(opts.PoolSize),
		pool.TTL(opts.PoolTTL),
		pool.Transport(opts.Transport),
	)

	rc := &rpcClient{
		opts: opts,
		pool: p,
		seq:  0,
	}
	rc.once.Store(false)

	c := client.Client(rc)

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

func (r *rpcClient) call(ctx context.Context, addr string, req client.Request, resp interface{}, opts client.CallOptions) error {
	msg := &transport.Message{
		Header: make(map[string]string),
	}

	md, ok := metadata.FromContext(ctx)
	if ok {
		for k, v := range md {
			// don't copy Micro-Topic header, that used for pub/sub
			// this fix case then client uses the same context that received in subscriber
			if k == "Micro-Topic" {
				continue
			}
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

	dOpts := []transport.DialOption{
		transport.WithStream(),
	}

	if opts.DialTimeout >= 0 {
		dOpts = append(dOpts, transport.WithTimeout(opts.DialTimeout))
	}

	c, err := r.pool.Get(addr, dOpts...)
	if err != nil {
		return errors.InternalServerError("go.micro.client", "connection error: %v", err)
	}

	seq := atomic.AddUint64(&r.seq, 1) - 1
	codec := newRpcCodec(msg, c, cf, "")

	rsp := &rpcResponse{
		socket: c,
		codec:  codec,
	}

	stream := &rpcStream{
		id:       fmt.Sprintf("%v", seq),
		context:  ctx,
		request:  req,
		response: rsp,
		codec:    codec,
		closed:   make(chan bool),
		release:  func(err error) { r.pool.Release(c, err) },
		sendEOS:  false,
	}
	// close the stream on exiting this function
	defer stream.Close()

	// wait for error response
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

	var grr error

	select {
	case err := <-ch:
		return err
	case <-ctx.Done():
		grr = errors.Timeout("go.micro.client", fmt.Sprintf("%v", ctx.Err()))
	}

	// set the stream error
	if grr != nil {
		stream.Lock()
		stream.err = grr
		stream.Unlock()

		return grr
	}

	return nil
}

func (r *rpcClient) stream(ctx context.Context, addr string, req client.Request, opts client.CallOptions) (client.Stream, error) {
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
	if opts.StreamTimeout > time.Duration(0) {
		msg.Header["Timeout"] = fmt.Sprintf("%d", opts.StreamTimeout)
	}
	// set the content type for the request
	msg.Header["Content-Type"] = req.ContentType()
	// set the accept header
	msg.Header["Accept"] = req.ContentType()

	cf, err := r.newCodec(req.ContentType())
	if err != nil {
		return nil, errors.InternalServerError("go.micro.client", err.Error())
	}

	dOpts := []transport.DialOption{
		transport.WithStream(),
	}

	if opts.DialTimeout >= 0 {
		dOpts = append(dOpts, transport.WithTimeout(opts.DialTimeout))
	}

	c, err := r.opts.Transport.Dial(addr, dOpts...)
	if err != nil {
		return nil, errors.InternalServerError("go.micro.client", "connection error: %v", err)
	}

	// increment the sequence number
	seq := atomic.AddUint64(&r.seq, 1) - 1
	id := fmt.Sprintf("%v", seq)

	// create codec with stream id
	codec := newRpcCodec(msg, c, cf, id)

	rsp := &rpcResponse{
		socket: c,
		codec:  codec,
	}

	// set request codec
	if r, ok := req.(*rpcRequest); ok {
		r.codec = codec
	}

	stream := &rpcStream{
		id:       id,
		context:  ctx,
		request:  req,
		response: rsp,
		codec:    codec,
		// used to close the stream
		closed: make(chan bool),
		// signal the end of stream,
		sendEOS: true,
		// release func
		release: func(err error) { c.Close() },
	}

	// wait for error response
	ch := make(chan error, 1)

	go func() {
		// send the first message
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
		// set the error
		stream.Lock()
		stream.err = grr
		stream.Unlock()

		// close the stream
		stream.Close()
		return nil, grr
	}

	return stream, nil
}

func (r *rpcClient) Init(opts ...client.Option) error {
	size := r.opts.PoolSize
	ttl := r.opts.PoolTTL
	tr := r.opts.Transport

	for _, o := range opts {
		o(&r.opts)
	}

	// update pool configuration if the options changed
	if size != r.opts.PoolSize || ttl != r.opts.PoolTTL || tr != r.opts.Transport {
		// close existing pool
		r.pool.Close()
		// create new pool
		r.pool = pool.NewPool(
			pool.Size(r.opts.PoolSize),
			pool.TTL(r.opts.PoolTTL),
			pool.Transport(r.opts.Transport),
		)
	}

	return nil
}

func (r *rpcClient) Options() client.Options {
	return r.opts
}

func (r *rpcClient) Call(ctx context.Context, request client.Request, response interface{}, opts ...client.CallOption) error {
	// make a copy of call opts
	callOpts := r.opts.CallOptions
	for _, opt := range opts {
		opt(&callOpts)
	}

	// check if we already have a deadline
	if d, ok := ctx.Deadline(); !ok {
		// no deadline so we create a new one
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, callOpts.RequestTimeout)
		defer cancel()
	} else {
		// got a deadline so no need to setup context
		// but we need to set the timeout we pass along
		remaining := d.Sub(time.Now())
		client.WithRequestTimeout(remaining)(&callOpts)
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

	// use the router passed as a call option, or fallback to the rpc clients router
	if callOpts.Router == nil {
		callOpts.Router = r.opts.Router
	}

	if callOpts.Selector == nil {
		callOpts.Selector = r.opts.Selector
	}

	// inject proxy address
	// TODO: don't even bother using Lookup/Select in this case
	if len(r.opts.Proxy) > 0 {
		callOpts.Address = []string{r.opts.Proxy}
	}

	// lookup the route to send the reques to
	// TODO apply any filtering here
	routes, err := r.opts.Lookup(ctx, request, callOpts)
	if err != nil {
		return errors.InternalServerError("go.micro.client", err.Error())
	}

	// balance the list of nodes
	next, err := callOpts.Selector.Select(routes)
	if err != nil {
		return err
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

		// get the next node
		node := next()

		// make the call
		err = rcall(ctx, node, request, response, callOpts)

		// record the result of the call to inform future routing decisions
		r.opts.Selector.Record(node, err)

		return err
	}

	// get the retries
	retries := callOpts.Retries

	// disable retries when using a proxy
	if len(r.opts.Proxy) > 0 {
		retries = 0
	}

	ch := make(chan error, retries+1)
	var gerr error

	for i := 0; i <= retries; i++ {
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

func (r *rpcClient) Stream(ctx context.Context, request client.Request, opts ...client.CallOption) (client.Stream, error) {
	// make a copy of call opts
	callOpts := r.opts.CallOptions
	for _, opt := range opts {
		opt(&callOpts)
	}

	// should we noop right here?
	select {
	case <-ctx.Done():
		return nil, errors.Timeout("go.micro.client", fmt.Sprintf("%v", ctx.Err()))
	default:
	}

	// use the router passed as a call option, or fallback to the rpc clients router
	if callOpts.Router == nil {
		callOpts.Router = r.opts.Router
	}

	if callOpts.Selector == nil {
		callOpts.Selector = r.opts.Selector
	}

	// inject proxy address
	// TODO: don't even bother using Lookup/Select in this case
	if len(r.opts.Proxy) > 0 {
		callOpts.Address = []string{r.opts.Proxy}
	}

	// lookup the route to send the reques to
	// TODO apply any filtering here
	routes, err := r.opts.Lookup(ctx, request, callOpts)
	if err != nil {
		return nil, errors.InternalServerError("go.micro.client", err.Error())
	}

	// balance the list of nodes
	next, err := callOpts.Selector.Select(routes)
	if err != nil {
		return nil, err
	}

	call := func(i int) (client.Stream, error) {
		// call backoff first. Someone may want an initial start delay
		t, err := callOpts.Backoff(ctx, request, i)
		if err != nil {
			return nil, errors.InternalServerError("go.micro.client", "backoff error: %v", err.Error())
		}

		// only sleep if greater than 0
		if t.Seconds() > 0 {
			time.Sleep(t)
		}

		// get the next node
		node := next()

		// perform the call
		stream, err := r.stream(ctx, node, request, callOpts)

		// record the result of the call to inform future routing decisions
		r.opts.Selector.Record(node, err)

		return stream, err
	}

	type response struct {
		stream client.Stream
		err    error
	}

	// get the retries
	retries := callOpts.Retries

	// disable retries when using a proxy
	if len(r.opts.Proxy) > 0 {
		retries = 0
	}

	ch := make(chan response, retries+1)
	var grr error

	for i := 0; i <= retries; i++ {
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

func (r *rpcClient) Publish(ctx context.Context, msg client.Message, opts ...client.PublishOption) error {
	options := client.PublishOptions{
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

	// get the exchange
	if len(options.Exchange) > 0 {
		topic = options.Exchange
	}

	// encode message body
	cf, err := r.newCodec(msg.ContentType())
	if err != nil {
		return errors.InternalServerError("go.micro.client", err.Error())
	}

	var body []byte

	// passed in raw data
	if d, ok := msg.Payload().(*raw.Frame); ok {
		body = d.Data
	} else {
		// new buffer
		b := buf.New(nil)

		if err := cf(b).Write(&codec.Message{
			Target: topic,
			Type:   codec.Event,
			Header: map[string]string{
				"Micro-Id":    id,
				"Micro-Topic": msg.Topic(),
			},
		}, msg.Payload()); err != nil {
			return errors.InternalServerError("go.micro.client", err.Error())
		}

		// set the body
		body = b.Bytes()
	}

	if !r.once.Load().(bool) {
		if err = r.opts.Broker.Connect(); err != nil {
			return errors.InternalServerError("go.micro.client", err.Error())
		}
		r.once.Store(true)
	}

	return r.opts.Broker.Publish(topic, &broker.Message{
		Header: md,
		Body:   body,
	}, broker.PublishContext(options.Context))
}

func (r *rpcClient) NewMessage(topic string, message interface{}, opts ...client.MessageOption) client.Message {
	return newMessage(topic, message, r.opts.ContentType, opts...)
}

func (r *rpcClient) NewRequest(service, method string, request interface{}, reqOpts ...client.RequestOption) client.Request {
	return newRequest(service, method, request, r.opts.ContentType, reqOpts...)
}

func (r *rpcClient) String() string {
	return "mucp"
}
