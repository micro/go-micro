package client

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"go-micro.dev/v4/broker"
	"go-micro.dev/v4/codec"
	raw "go-micro.dev/v4/codec/bytes"
	merrors "go-micro.dev/v4/errors"
	log "go-micro.dev/v4/logger"
	"go-micro.dev/v4/metadata"
	"go-micro.dev/v4/registry"
	"go-micro.dev/v4/selector"
	"go-micro.dev/v4/transport"
	"go-micro.dev/v4/transport/headers"
	"go-micro.dev/v4/util/buf"
	"go-micro.dev/v4/util/net"
	"go-micro.dev/v4/util/pool"
)

const (
	packageID = "go.micro.client"
)

type rpcClient struct {
	seq  uint64
	once atomic.Value
	opts Options
	pool pool.Pool

	mu sync.RWMutex
}

func newRPCClient(opt ...Option) Client {
	opts := NewOptions(opt...)

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

	return nil, fmt.Errorf("unsupported Content-Type: %s", contentType)
}

func (r *rpcClient) call(ctx context.Context, node *registry.Node, req Request, resp interface{}, opts CallOptions) error {
	address := node.Address
	logger := r.Options().Logger

	msg := &transport.Message{
		Header: make(map[string]string),
	}

	md, ok := metadata.FromContext(ctx)
	if ok {
		for k, v := range md {
			// Don't copy Micro-Topic header, that is used for pub/sub
			// this is fixes the case when the client uses the same context that
			// is received in the subscriber.
			if k == headers.Message {
				continue
			}

			msg.Header[k] = v
		}
	}

	// Set connection timeout for single requests to the server. Should be > 0
	// as otherwise requests can't be made.
	cTimeout := opts.ConnectionTimeout
	if cTimeout == 0 {
		logger.Log(log.DebugLevel, "connection timeout was set to 0, overridng to default connection timeout")

		cTimeout = DefaultConnectionTimeout
	}

	// set timeout in nanoseconds
	msg.Header["Timeout"] = fmt.Sprintf("%d", cTimeout)
	// set the content type for the request
	msg.Header["Content-Type"] = req.ContentType()
	// set the accept header
	msg.Header["Accept"] = req.ContentType()

	// setup old protocol
	reqCodec := setupProtocol(msg, node)

	// no codec specified
	if reqCodec == nil {
		var err error
		reqCodec, err = r.newCodec(req.ContentType())

		if err != nil {
			return merrors.InternalServerError("go.micro.client", err.Error())
		}
	}

	dOpts := []transport.DialOption{
		transport.WithStream(),
	}

	if opts.DialTimeout >= 0 {
		dOpts = append(dOpts, transport.WithTimeout(opts.DialTimeout))
	}

	if opts.ConnClose {
		dOpts = append(dOpts, transport.WithConnClose())
	}

	c, err := r.pool.Get(address, dOpts...)
	if err != nil {
		return merrors.InternalServerError("go.micro.client", "connection error: %v", err)
	}

	seq := atomic.AddUint64(&r.seq, 1) - 1
	codec := newRPCCodec(msg, c, reqCodec, "")

	rsp := &rpcResponse{
		socket: c,
		codec:  codec,
	}

	releaseFunc := func(err error) {
		if err = r.pool.Release(c, err); err != nil {
			logger.Log(log.ErrorLevel, "failed to release pool", err)
		}
	}

	stream := &rpcStream{
		id:       fmt.Sprintf("%v", seq),
		context:  ctx,
		request:  req,
		response: rsp,
		codec:    codec,
		closed:   make(chan bool),
		close:    opts.ConnClose,
		release:  releaseFunc,
		sendEOS:  false,
	}

	// close the stream on exiting this function
	defer func() {
		if err := stream.Close(); err != nil {
			logger.Log(log.ErrorLevel, "failed to close stream", err)
		}
	}()

	// wait for error response
	ch := make(chan error, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				ch <- merrors.InternalServerError("go.micro.client", "panic recovered: %v", r)
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
	case <-time.After(cTimeout):
		grr = merrors.Timeout("go.micro.client", fmt.Sprintf("%v", ctx.Err()))
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

func (r *rpcClient) stream(ctx context.Context, node *registry.Node, req Request, opts CallOptions) (Stream, error) {
	address := node.Address
	logger := r.Options().Logger

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

	// set old codecs
	nCodec := setupProtocol(msg, node)

	// no codec specified
	if nCodec == nil {
		var err error

		nCodec, err = r.newCodec(req.ContentType())
		if err != nil {
			return nil, merrors.InternalServerError("go.micro.client", err.Error())
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
		return nil, merrors.InternalServerError("go.micro.client", "connection error: %v", err)
	}

	// increment the sequence number
	seq := atomic.AddUint64(&r.seq, 1) - 1
	id := fmt.Sprintf("%v", seq)

	// create codec with stream id
	codec := newRPCCodec(msg, c, nCodec, id)

	rsp := &rpcResponse{
		socket: c,
		codec:  codec,
	}

	// set request codec
	if r, ok := req.(*rpcRequest); ok {
		r.codec = codec
	}

	releaseFunc := func(_ error) {
		if err = c.Close(); err != nil {
			logger.Log(log.ErrorLevel, err)
		}
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
		release: releaseFunc,
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
		grr = merrors.Timeout("go.micro.client", fmt.Sprintf("%v", ctx.Err()))
	}

	if grr != nil {
		// set the error
		stream.Lock()
		stream.err = grr
		stream.Unlock()

		// close the stream
		if err := stream.Close(); err != nil {
			logger.Logf(log.ErrorLevel, "failed to close stream: %v", err)
		}

		return nil, grr
	}

	return stream, nil
}

func (r *rpcClient) Init(opts ...Option) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	size := r.opts.PoolSize
	ttl := r.opts.PoolTTL
	tr := r.opts.Transport

	for _, o := range opts {
		o(&r.opts)
	}

	// update pool configuration if the options changed
	if size != r.opts.PoolSize || ttl != r.opts.PoolTTL || tr != r.opts.Transport {
		// close existing pool
		if err := r.pool.Close(); err != nil {
			return errors.Wrap(err, "failed to close pool")
		}

		// create new pool
		r.pool = pool.NewPool(
			pool.Size(r.opts.PoolSize),
			pool.TTL(r.opts.PoolTTL),
			pool.Transport(r.opts.Transport),
		)
	}

	return nil
}

// Options retrives the options.
func (r *rpcClient) Options() Options {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.opts
}

// next returns an iterator for the next nodes to call.
func (r *rpcClient) next(request Request, opts CallOptions) (selector.Next, error) {
	// try get the proxy
	service, address, _ := net.Proxy(request.Service(), opts.Address)

	// return remote address
	if len(address) > 0 {
		nodes := make([]*registry.Node, len(address))

		for i, addr := range address {
			nodes[i] = &registry.Node{
				Address: addr,
				// Set the protocol
				Metadata: map[string]string{
					"protocol": "mucp",
				},
			}
		}

		// crude return method
		return func() (*registry.Node, error) {
			return nodes[time.Now().Unix()%int64(len(nodes))], nil
		}, nil
	}

	// get next nodes from the selector
	next, err := r.opts.Selector.Select(service, opts.SelectOptions...)
	if err != nil {
		if errors.Is(err, selector.ErrNotFound) {
			return nil, merrors.InternalServerError("go.micro.client", "service %s: %s", service, err.Error())
		}

		return nil, merrors.InternalServerError("go.micro.client", "error selecting %s node: %s", service, err.Error())
	}

	return next, nil
}

func (r *rpcClient) Call(ctx context.Context, request Request, response interface{}, opts ...CallOption) error {
	// TODO: further validate these mutex locks. full lock would prevent
	// parallel calls. Maybe we can set individual locks for secctions.
	r.mu.RLock()
	defer r.mu.RUnlock()

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
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, callOpts.RequestTimeout)

		defer cancel()
	} else {
		// got a deadline so no need to setup context
		// but we need to set the timeout we pass along
		opt := WithRequestTimeout(time.Until(d))
		opt(&callOpts)
	}

	// should we noop right here?
	select {
	case <-ctx.Done():
		return merrors.Timeout("go.micro.client", fmt.Sprintf("%v", ctx.Err()))
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
			return merrors.InternalServerError("go.micro.client", "backoff error: %v", err.Error())
		}

		// only sleep if greater than 0
		if t.Seconds() > 0 {
			time.Sleep(t)
		}

		// select next node
		node, err := next()
		service := request.Service()

		if err != nil {
			if errors.Is(err, selector.ErrNotFound) {
				return merrors.InternalServerError("go.micro.client", "service %s: %s", service, err.Error())
			}

			return merrors.InternalServerError("go.micro.client", "error getting next %s node: %s", service, err.Error())
		}

		// make the call
		err = rcall(ctx, node, request, response, callOpts)
		r.opts.Selector.Mark(service, node, err)

		return err
	}

	// get the retries
	retries := callOpts.Retries

	// disable retries when using a proxy
	// Note: I don't see why we should disable retries for proxies, so commenting out.
	// if _, _, ok := net.Proxy(request.Service(), callOpts.Address); ok {
	// 	retries = 0
	// }

	ch := make(chan error, retries+1)

	var gerr error

	for i := 0; i <= retries; i++ {
		go func(i int) {
			ch <- call(i)
		}(i)

		select {
		case <-ctx.Done():
			return merrors.Timeout("go.micro.client", fmt.Sprintf("call timeout: %v", ctx.Err()))
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

			r.opts.Logger.Logf(log.DebugLevel, "Retrying request. Previous attempt failed with: %v", err)

			gerr = err
		}
	}

	return gerr
}

func (r *rpcClient) Stream(ctx context.Context, request Request, opts ...CallOption) (Stream, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// make a copy of call opts
	callOpts := r.opts.CallOptions
	for _, opt := range opts {
		opt(&callOpts)
	}

	next, err := r.next(request, callOpts)
	if err != nil {
		return nil, err
	}

	select {
	case <-ctx.Done():
		return nil, merrors.Timeout("go.micro.client", fmt.Sprintf("%v", ctx.Err()))
	default:
	}

	call := func(i int) (Stream, error) {
		// call backoff first. Someone may want an initial start delay
		t, err := callOpts.Backoff(ctx, request, i)
		if err != nil {
			return nil, merrors.InternalServerError("go.micro.client", "backoff error: %v", err.Error())
		}

		// only sleep if greater than 0
		if t.Seconds() > 0 {
			time.Sleep(t)
		}

		node, err := next()
		service := request.Service()

		if err != nil {
			if errors.Is(err, selector.ErrNotFound) {
				return nil, merrors.InternalServerError("go.micro.client", "service %s: %s", service, err.Error())
			}

			return nil, merrors.InternalServerError("go.micro.client", "error getting next %s node: %s", service, err.Error())
		}

		stream, err := r.stream(ctx, node, request, callOpts)
		r.opts.Selector.Mark(service, node, err)

		return stream, err
	}

	type response struct {
		stream Stream
		err    error
	}

	// get the retries
	retries := callOpts.Retries

	// disable retries when using a proxy
	if _, _, ok := net.Proxy(request.Service(), callOpts.Address); ok {
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
			return nil, merrors.Timeout("go.micro.client", fmt.Sprintf("call timeout: %v", ctx.Err()))
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

	metadata, ok := metadata.FromContext(ctx)
	if !ok {
		metadata = make(map[string]string)
	}

	id := uuid.New().String()
	metadata["Content-Type"] = msg.ContentType()
	metadata[headers.Message] = msg.Topic()
	metadata[headers.ID] = id

	// set the topic
	topic := msg.Topic()

	// get the exchange
	if len(options.Exchange) > 0 {
		topic = options.Exchange
	}

	// encode message body
	cf, err := r.newCodec(msg.ContentType())
	if err != nil {
		return merrors.InternalServerError(packageID, err.Error())
	}

	var body []byte

	// passed in raw data
	if d, ok := msg.Payload().(*raw.Frame); ok {
		body = d.Data
	} else {
		b := buf.New(nil)

		if err = cf(b).Write(&codec.Message{
			Target: topic,
			Type:   codec.Event,
			Header: map[string]string{
				headers.ID:      id,
				headers.Message: msg.Topic(),
			},
		}, msg.Payload()); err != nil {
			return merrors.InternalServerError(packageID, err.Error())
		}

		// set the body
		body = b.Bytes()
	}

	l, ok := r.once.Load().(bool)
	if !ok {
		return fmt.Errorf("failed to cast to bool")
	}

	if !l {
		if err = r.opts.Broker.Connect(); err != nil {
			return merrors.InternalServerError(packageID, err.Error())
		}

		r.once.Store(true)
	}

	return r.opts.Broker.Publish(topic, &broker.Message{
		Header: metadata,
		Body:   body,
	}, broker.PublishContext(options.Context))
}

func (r *rpcClient) NewMessage(topic string, message interface{}, opts ...MessageOption) Message {
	return newMessage(topic, message, r.opts.ContentType, opts...)
}

func (r *rpcClient) NewRequest(service, method string, request interface{}, reqOpts ...RequestOption) Request {
	return newRequest(service, method, request, r.opts.ContentType, reqOpts...)
}

func (r *rpcClient) String() string {
	return "mucp"
}
