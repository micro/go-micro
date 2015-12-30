package client

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/micro/go-micro/broker"
	"github.com/micro/go-micro/codec"
	c "github.com/micro/go-micro/context"
	"github.com/micro/go-micro/errors"
	"github.com/micro/go-micro/registry"
	"github.com/micro/go-micro/selector"
	"github.com/micro/go-micro/transport"

	"golang.org/x/net/context"
)

type rpcClient struct {
	once sync.Once
	opts options
}

func newRpcClient(opt ...Option) Client {
	var once sync.Once

	opts := options{
		codecs: make(map[string]codec.NewCodec),
	}

	for _, o := range opt {
		o(&opts)
	}

	if len(opts.contentType) == 0 {
		opts.contentType = defaultContentType
	}

	if opts.broker == nil {
		opts.broker = broker.DefaultBroker
	}

	if opts.registry == nil {
		opts.registry = registry.DefaultRegistry
	}

	if opts.selector == nil {
		opts.selector = selector.NewSelector(
			selector.Registry(opts.registry),
		)
	}

	if opts.transport == nil {
		opts.transport = transport.DefaultTransport
	}

	rc := &rpcClient{
		once: once,
		opts: opts,
	}

	c := Client(rc)

	// wrap in reverse
	for i := len(opts.wrappers); i > 0; i-- {
		c = opts.wrappers[i-1](c)
	}

	return c
}

func (r *rpcClient) newCodec(contentType string) (codec.NewCodec, error) {
	if c, ok := r.opts.codecs[contentType]; ok {
		return c, nil
	}
	if cf, ok := defaultCodecs[contentType]; ok {
		return cf, nil
	}
	return nil, fmt.Errorf("Unsupported Content-Type: %s", contentType)
}

func (r *rpcClient) call(ctx context.Context, address string, request Request, response interface{}) error {
	msg := &transport.Message{
		Header: make(map[string]string),
	}

	md, ok := c.GetMetadata(ctx)
	if ok {
		for k, v := range md {
			msg.Header[k] = v
		}
	}

	msg.Header["Content-Type"] = request.ContentType()

	cf, err := r.newCodec(request.ContentType())
	if err != nil {
		return errors.InternalServerError("go.micro.client", err.Error())
	}

	c, err := r.opts.transport.Dial(address)
	if err != nil {
		return errors.InternalServerError("go.micro.client", fmt.Sprintf("Error sending request: %v", err))
	}
	defer c.Close()

	client := newClientWithCodec(newRpcPlusCodec(msg, c, cf))
	err = client.Call(ctx, request.Service(), request.Method(), request.Request(), response)
	if err != nil {
		return err
	}
	return client.Close()
}

func (r *rpcClient) stream(ctx context.Context, address string, req Request) (Streamer, error) {
	msg := &transport.Message{
		Header: make(map[string]string),
	}

	md, ok := c.GetMetadata(ctx)
	if ok {
		for k, v := range md {
			msg.Header[k] = v
		}
	}

	msg.Header["Content-Type"] = req.ContentType()

	cf, err := r.newCodec(req.ContentType())
	if err != nil {
		return nil, errors.InternalServerError("go.micro.client", err.Error())
	}

	c, err := r.opts.transport.Dial(address, transport.WithStream())
	if err != nil {
		return nil, errors.InternalServerError("go.micro.client", fmt.Sprintf("Error sending request: %v", err))
	}

	var once sync.Once
	stream := &rpcStream{
		context: ctx,
		request: req,
		once:    once,
		closed:  make(chan bool),
		codec:   newRpcPlusCodec(msg, c, cf),
	}

	err = stream.Send(req.Request())
	return stream, err
}

func (r *rpcClient) CallRemote(ctx context.Context, address string, request Request, response interface{}, opts ...CallOption) error {
	return r.call(ctx, address, request, response)
}

func (r *rpcClient) Call(ctx context.Context, request Request, response interface{}, opts ...CallOption) error {
	var copts callOptions
	for _, opt := range opts {
		opt(&copts)
	}

	next, err := r.opts.selector.Select(request.Service(), copts.selectOptions...)
	if err != nil && err == selector.ErrNotFound {
		return errors.NotFound("go.micro.client", err.Error())
	} else if err != nil {
		return errors.InternalServerError("go.micro.client", err.Error())
	}

	node, err := next()
	if err != nil && err == selector.ErrNotFound {
		return errors.NotFound("go.micro.client", err.Error())
	} else if err != nil {
		return errors.InternalServerError("go.micro.client", err.Error())
	}

	address := node.Address
	if node.Port > 0 {
		address = fmt.Sprintf("%s:%d", address, node.Port)
	}

	err = r.call(ctx, address, request, response)
	r.opts.selector.Mark(request.Service(), node, err)
	return err
}

func (r *rpcClient) StreamRemote(ctx context.Context, address string, request Request, opts ...CallOption) (Streamer, error) {
	return r.stream(ctx, address, request)
}

func (r *rpcClient) Stream(ctx context.Context, request Request, opts ...CallOption) (Streamer, error) {
	var copts callOptions
	for _, opt := range opts {
		opt(&copts)
	}

	next, err := r.opts.selector.Select(request.Service(), copts.selectOptions...)
	if err != nil && err == selector.ErrNotFound {
		return nil, errors.NotFound("go.micro.client", err.Error())
	} else if err != nil {
		return nil, errors.InternalServerError("go.micro.client", err.Error())
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

	stream, err := r.stream(ctx, address, request)
	r.opts.selector.Mark(request.Service(), node, err)
	return stream, err
}

func (r *rpcClient) Publish(ctx context.Context, p Publication, opts ...PublishOption) error {
	md, ok := c.GetMetadata(ctx)
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
		r.opts.broker.Connect()
	})

	return r.opts.broker.Publish(p.Topic(), &broker.Message{
		Header: md,
		Body:   b.Bytes(),
	})
}

func (r *rpcClient) NewPublication(topic string, message interface{}) Publication {
	return newRpcPublication(topic, message, r.opts.contentType)
}

func (r *rpcClient) NewProtoPublication(topic string, message interface{}) Publication {
	return newRpcPublication(topic, message, "application/octet-stream")
}
func (r *rpcClient) NewRequest(service, method string, request interface{}, reqOpts ...RequestOption) Request {
	return newRpcRequest(service, method, request, r.opts.contentType, reqOpts...)
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
