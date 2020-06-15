// Package grpc provides a gRPC client
package grpc

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"reflect"
	"strings"
	"sync/atomic"
	"time"

	"github.com/micro/go-micro/v2/broker"
	"github.com/micro/go-micro/v2/client"
	"github.com/micro/go-micro/v2/client/selector"
	raw "github.com/micro/go-micro/v2/codec/bytes"
	"github.com/micro/go-micro/v2/errors"
	"github.com/micro/go-micro/v2/metadata"
	"github.com/micro/go-micro/v2/registry"
	pnet "github.com/micro/go-micro/v2/util/net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/encoding"
	gmetadata "google.golang.org/grpc/metadata"
)

type grpcClient struct {
	opts client.Options
	pool *pool
	once atomic.Value
}

func init() {
	encoding.RegisterCodec(wrapCodec{jsonCodec{}})
	encoding.RegisterCodec(wrapCodec{protoCodec{}})
	encoding.RegisterCodec(wrapCodec{bytesCodec{}})
}

// secure returns the dial option for whether its a secure or insecure connection
func (g *grpcClient) secure(addr string) grpc.DialOption {
	// first we check if theres'a  tls config
	if g.opts.Context != nil {
		if v := g.opts.Context.Value(tlsAuth{}); v != nil {
			tls := v.(*tls.Config)
			creds := credentials.NewTLS(tls)
			// return tls config if it exists
			return grpc.WithTransportCredentials(creds)
		}
	}

	// default config
	tlsConfig := &tls.Config{}
	defaultCreds := grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig))

	// check if the address is prepended with https
	if strings.HasPrefix(addr, "https://") {
		return defaultCreds
	}

	// if no port is specified or port is 443 default to tls
	_, port, err := net.SplitHostPort(addr)
	// assuming with no port its going to be secured
	if port == "443" {
		return defaultCreds
	} else if err != nil && strings.Contains(err.Error(), "missing port in address") {
		return defaultCreds
	}

	// other fallback to insecure
	return grpc.WithInsecure()
}

func (g *grpcClient) next(request client.Request, opts client.CallOptions) (selector.Next, error) {
	service, address, _ := pnet.Proxy(request.Service(), opts.Address)

	// return remote address
	if len(address) > 0 {
		return func() (*registry.Node, error) {
			return &registry.Node{
				Address: address[0],
			}, nil
		}, nil
	}

	// get next nodes from the selector
	next, err := g.opts.Selector.Select(service, opts.SelectOptions...)
	if err != nil {
		if err == selector.ErrNotFound {
			return nil, errors.InternalServerError("go.micro.client", "service %s: %s", service, err.Error())
		}
		return nil, errors.InternalServerError("go.micro.client", "error selecting %s node: %s", service, err.Error())
	}

	return next, nil
}

func (g *grpcClient) call(ctx context.Context, node *registry.Node, req client.Request, rsp interface{}, opts client.CallOptions) error {
	var header map[string]string

	address := node.Address

	header = make(map[string]string)
	if md, ok := metadata.FromContext(ctx); ok {
		header = make(map[string]string, len(md))
		for k, v := range md {
			header[strings.ToLower(k)] = v
		}
	} else {
		header = make(map[string]string)
	}

	// set timeout in nanoseconds
	header["timeout"] = fmt.Sprintf("%d", opts.RequestTimeout)
	// set the content type for the request
	header["x-content-type"] = req.ContentType()

	md := gmetadata.New(header)
	ctx = gmetadata.NewOutgoingContext(ctx, md)

	cf, err := g.newGRPCCodec(req.ContentType())
	if err != nil {
		return errors.InternalServerError("go.micro.client", err.Error())
	}

	maxRecvMsgSize := g.maxRecvMsgSizeValue()
	maxSendMsgSize := g.maxSendMsgSizeValue()

	var grr error

	grpcDialOptions := []grpc.DialOption{
		grpc.WithTimeout(opts.DialTimeout),
		g.secure(address),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(maxRecvMsgSize),
			grpc.MaxCallSendMsgSize(maxSendMsgSize),
		),
	}

	if opts := g.getGrpcDialOptions(); opts != nil {
		grpcDialOptions = append(grpcDialOptions, opts...)
	}

	cc, err := g.pool.getConn(address, grpcDialOptions...)
	if err != nil {
		return errors.InternalServerError("go.micro.client", fmt.Sprintf("Error sending request: %v", err))
	}
	defer func() {
		// defer execution of release
		g.pool.release(address, cc, grr)
	}()

	ch := make(chan error, 1)

	go func() {
		grpcCallOptions := []grpc.CallOption{
			grpc.ForceCodec(cf),
			grpc.CallContentSubtype(cf.Name())}
		if opts := g.getGrpcCallOptions(); opts != nil {
			grpcCallOptions = append(grpcCallOptions, opts...)
		}
		err := cc.Invoke(ctx, methodToGRPC(req.Service(), req.Endpoint()), req.Body(), rsp, grpcCallOptions...)
		ch <- microError(err)
	}()

	select {
	case err := <-ch:
		grr = err
	case <-ctx.Done():
		grr = errors.Timeout("go.micro.client", "%v", ctx.Err())
	}

	return grr
}

func (g *grpcClient) stream(ctx context.Context, node *registry.Node, req client.Request, rsp interface{}, opts client.CallOptions) error {
	var header map[string]string

	address := node.Address

	if md, ok := metadata.FromContext(ctx); ok {
		header = make(map[string]string, len(md))
		for k, v := range md {
			header[k] = v
		}
	} else {
		header = make(map[string]string)
	}

	// set timeout in nanoseconds
	if opts.StreamTimeout > time.Duration(0) {
		header["timeout"] = fmt.Sprintf("%d", opts.StreamTimeout)
	}
	// set the content type for the request
	header["x-content-type"] = req.ContentType()

	md := gmetadata.New(header)
	ctx = gmetadata.NewOutgoingContext(ctx, md)

	cf, err := g.newGRPCCodec(req.ContentType())
	if err != nil {
		return errors.InternalServerError("go.micro.client", err.Error())
	}

	var dialCtx context.Context
	var cancel context.CancelFunc
	if opts.DialTimeout >= 0 {
		dialCtx, cancel = context.WithTimeout(ctx, opts.DialTimeout)
	} else {
		dialCtx, cancel = context.WithCancel(ctx)
	}
	defer cancel()

	wc := wrapCodec{cf}

	grpcDialOptions := []grpc.DialOption{
		grpc.WithTimeout(opts.DialTimeout),
		g.secure(address),
	}

	if opts := g.getGrpcDialOptions(); opts != nil {
		grpcDialOptions = append(grpcDialOptions, opts...)
	}

	cc, err := grpc.DialContext(dialCtx, address, grpcDialOptions...)
	if err != nil {
		return errors.InternalServerError("go.micro.client", fmt.Sprintf("Error sending request: %v", err))
	}

	desc := &grpc.StreamDesc{
		StreamName:    req.Service() + req.Endpoint(),
		ClientStreams: true,
		ServerStreams: true,
	}

	grpcCallOptions := []grpc.CallOption{
		grpc.ForceCodec(wc),
		grpc.CallContentSubtype(cf.Name()),
	}
	if opts := g.getGrpcCallOptions(); opts != nil {
		grpcCallOptions = append(grpcCallOptions, opts...)
	}

	// create a new cancelling context
	newCtx, cancel := context.WithCancel(ctx)

	st, err := cc.NewStream(newCtx, desc, methodToGRPC(req.Service(), req.Endpoint()), grpcCallOptions...)
	if err != nil {
		// we need to cleanup as we dialled and created a context
		// cancel the context
		cancel()
		// close the connection
		cc.Close()
		// now return the error
		return errors.InternalServerError("go.micro.client", fmt.Sprintf("Error creating stream: %v", err))
	}

	codec := &grpcCodec{
		s: st,
		c: wc,
	}

	// set request codec
	if r, ok := req.(*grpcRequest); ok {
		r.codec = codec
	}

	// setup the stream response
	stream := &grpcStream{
		context: ctx,
		request: req,
		response: &response{
			conn:   cc,
			stream: st,
			codec:  cf,
			gcodec: codec,
		},
		stream: st,
		conn:   cc,
		cancel: cancel,
	}

	// set the stream as the response
	val := reflect.ValueOf(rsp).Elem()
	val.Set(reflect.ValueOf(stream).Elem())
	return nil
}

func (g *grpcClient) poolMaxStreams() int {
	if g.opts.Context == nil {
		return DefaultPoolMaxStreams
	}
	v := g.opts.Context.Value(poolMaxStreams{})
	if v == nil {
		return DefaultPoolMaxStreams
	}
	return v.(int)
}

func (g *grpcClient) poolMaxIdle() int {
	if g.opts.Context == nil {
		return DefaultPoolMaxIdle
	}
	v := g.opts.Context.Value(poolMaxIdle{})
	if v == nil {
		return DefaultPoolMaxIdle
	}
	return v.(int)
}

func (g *grpcClient) maxRecvMsgSizeValue() int {
	if g.opts.Context == nil {
		return DefaultMaxRecvMsgSize
	}
	v := g.opts.Context.Value(maxRecvMsgSizeKey{})
	if v == nil {
		return DefaultMaxRecvMsgSize
	}
	return v.(int)
}

func (g *grpcClient) maxSendMsgSizeValue() int {
	if g.opts.Context == nil {
		return DefaultMaxSendMsgSize
	}
	v := g.opts.Context.Value(maxSendMsgSizeKey{})
	if v == nil {
		return DefaultMaxSendMsgSize
	}
	return v.(int)
}

func (g *grpcClient) newGRPCCodec(contentType string) (encoding.Codec, error) {
	codecs := make(map[string]encoding.Codec)
	if g.opts.Context != nil {
		if v := g.opts.Context.Value(codecsKey{}); v != nil {
			codecs = v.(map[string]encoding.Codec)
		}
	}
	if c, ok := codecs[contentType]; ok {
		return wrapCodec{c}, nil
	}
	if c, ok := defaultGRPCCodecs[contentType]; ok {
		return wrapCodec{c}, nil
	}
	return nil, fmt.Errorf("Unsupported Content-Type: %s", contentType)
}

func (g *grpcClient) Init(opts ...client.Option) error {
	size := g.opts.PoolSize
	ttl := g.opts.PoolTTL

	for _, o := range opts {
		o(&g.opts)
	}

	// update pool configuration if the options changed
	if size != g.opts.PoolSize || ttl != g.opts.PoolTTL {
		g.pool.Lock()
		g.pool.size = g.opts.PoolSize
		g.pool.ttl = int64(g.opts.PoolTTL.Seconds())
		g.pool.Unlock()
	}

	return nil
}

func (g *grpcClient) Options() client.Options {
	return g.opts
}

func (g *grpcClient) NewMessage(topic string, msg interface{}, opts ...client.MessageOption) client.Message {
	return newGRPCEvent(topic, msg, g.opts.ContentType, opts...)
}

func (g *grpcClient) NewRequest(service, method string, req interface{}, reqOpts ...client.RequestOption) client.Request {
	return newGRPCRequest(service, method, req, g.opts.ContentType, reqOpts...)
}

func (g *grpcClient) Call(ctx context.Context, req client.Request, rsp interface{}, opts ...client.CallOption) error {
	if req == nil {
		return errors.InternalServerError("go.micro.client", "req is nil")
	} else if rsp == nil {
		return errors.InternalServerError("go.micro.client", "rsp is nil")
	}
	// make a copy of call opts
	callOpts := g.opts.CallOptions
	for _, opt := range opts {
		opt(&callOpts)
	}

	next, err := g.next(req, callOpts)
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
		opt := client.WithRequestTimeout(time.Until(d))
		opt(&callOpts)
	}

	// should we noop right here?
	select {
	case <-ctx.Done():
		return errors.New("go.micro.client", fmt.Sprintf("%v", ctx.Err()), 408)
	default:
	}

	// make copy of call method
	gcall := g.call

	// wrap the call in reverse
	for i := len(callOpts.CallWrappers); i > 0; i-- {
		gcall = callOpts.CallWrappers[i-1](gcall)
	}

	// return errors.New("go.micro.client", "request timeout", 408)
	call := func(i int) error {
		// call backoff first. Someone may want an initial start delay
		t, err := callOpts.Backoff(ctx, req, i)
		if err != nil {
			return errors.InternalServerError("go.micro.client", err.Error())
		}

		// only sleep if greater than 0
		if t.Seconds() > 0 {
			time.Sleep(t)
		}

		// select next node
		node, err := next()
		service := req.Service()
		if err != nil {
			if err == selector.ErrNotFound {
				return errors.InternalServerError("go.micro.client", "service %s: %s", service, err.Error())
			}
			return errors.InternalServerError("go.micro.client", "error selecting %s node: %s", service, err.Error())
		}

		// make the call
		err = gcall(ctx, node, req, rsp, callOpts)
		g.opts.Selector.Mark(service, node, err)
		if verr, ok := err.(*errors.Error); ok {
			return verr
		}

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
			return errors.New("go.micro.client", fmt.Sprintf("%v", ctx.Err()), 408)
		case err := <-ch:
			// if the call succeeded lets bail early
			if err == nil {
				return nil
			}

			retry, rerr := callOpts.Retry(ctx, req, i, err)
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

func (g *grpcClient) Stream(ctx context.Context, req client.Request, opts ...client.CallOption) (client.Stream, error) {
	// make a copy of call opts
	callOpts := g.opts.CallOptions
	for _, opt := range opts {
		opt(&callOpts)
	}

	next, err := g.next(req, callOpts)
	if err != nil {
		return nil, err
	}

	// #200 - streams shouldn't have a request timeout set on the context

	// should we noop right here?
	select {
	case <-ctx.Done():
		return nil, errors.New("go.micro.client", fmt.Sprintf("%v", ctx.Err()), 408)
	default:
	}

	// make a copy of stream
	gstream := g.stream

	// wrap the call in reverse
	for i := len(callOpts.CallWrappers); i > 0; i-- {
		gstream = callOpts.CallWrappers[i-1](gstream)
	}

	call := func(i int) (client.Stream, error) {
		// call backoff first. Someone may want an initial start delay
		t, err := callOpts.Backoff(ctx, req, i)
		if err != nil {
			return nil, errors.InternalServerError("go.micro.client", err.Error())
		}

		// only sleep if greater than 0
		if t.Seconds() > 0 {
			time.Sleep(t)
		}

		node, err := next()
		service := req.Service()
		if err != nil {
			if err == selector.ErrNotFound {
				return nil, errors.InternalServerError("go.micro.client", "service %s: %s", service, err.Error())
			}
			return nil, errors.InternalServerError("go.micro.client", "error selecting %s node: %s", service, err.Error())
		}

		// make the call
		stream := &grpcStream{}
		err = g.stream(ctx, node, req, stream, callOpts)

		g.opts.Selector.Mark(service, node, err)
		return stream, err
	}

	type response struct {
		stream client.Stream
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
			return nil, errors.New("go.micro.client", fmt.Sprintf("%v", ctx.Err()), 408)
		case rsp := <-ch:
			// if the call succeeded lets bail early
			if rsp.err == nil {
				return rsp.stream, nil
			}

			retry, rerr := callOpts.Retry(ctx, req, i, err)
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

func (g *grpcClient) Publish(ctx context.Context, p client.Message, opts ...client.PublishOption) error {
	var options client.PublishOptions
	for _, o := range opts {
		o(&options)
	}

	md, ok := metadata.FromContext(ctx)
	if !ok {
		md = make(map[string]string)
	}
	md["Content-Type"] = p.ContentType()
	md["Micro-Topic"] = p.Topic()

	cf, err := g.newGRPCCodec(p.ContentType())
	if err != nil {
		return errors.InternalServerError("go.micro.client", err.Error())
	}

	var body []byte

	// passed in raw data
	if d, ok := p.Payload().(*raw.Frame); ok {
		body = d.Data
	} else {
		// set the body
		b, err := cf.Marshal(p.Payload())
		if err != nil {
			return errors.InternalServerError("go.micro.client", err.Error())
		}
		body = b
	}

	if !g.once.Load().(bool) {
		if err = g.opts.Broker.Connect(); err != nil {
			return errors.InternalServerError("go.micro.client", err.Error())
		}
		g.once.Store(true)
	}

	topic := p.Topic()

	// get the exchange
	if len(options.Exchange) > 0 {
		topic = options.Exchange
	}

	return g.opts.Broker.Publish(topic, &broker.Message{
		Header: md,
		Body:   body,
	}, broker.PublishContext(options.Context))
}

func (g *grpcClient) String() string {
	return "grpc"
}

func (g *grpcClient) getGrpcDialOptions() []grpc.DialOption {
	if g.opts.CallOptions.Context == nil {
		return nil
	}

	v := g.opts.CallOptions.Context.Value(grpcDialOptions{})

	if v == nil {
		return nil
	}

	opts, ok := v.([]grpc.DialOption)

	if !ok {
		return nil
	}

	return opts
}

func (g *grpcClient) getGrpcCallOptions() []grpc.CallOption {
	if g.opts.CallOptions.Context == nil {
		return nil
	}

	v := g.opts.CallOptions.Context.Value(grpcCallOptions{})

	if v == nil {
		return nil
	}

	opts, ok := v.([]grpc.CallOption)

	if !ok {
		return nil
	}

	return opts
}

func newClient(opts ...client.Option) client.Client {
	options := client.NewOptions()
	// default content type for grpc
	options.ContentType = "application/grpc+proto"

	for _, o := range opts {
		o(&options)
	}

	rc := &grpcClient{
		opts: options,
	}
	rc.once.Store(false)

	rc.pool = newPool(options.PoolSize, options.PoolTTL, rc.poolMaxIdle(), rc.poolMaxStreams())

	c := client.Client(rc)

	// wrap in reverse
	for i := len(options.Wrappers); i > 0; i-- {
		c = options.Wrappers[i-1](c)
	}

	return c
}

func NewClient(opts ...client.Option) client.Client {
	return newClient(opts...)
}
