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

	if opts.transport == nil {
		opts.transport = transport.DefaultTransport
	}

	if opts.broker == nil {
		opts.broker = broker.DefaultBroker
	}

	if opts.selector == nil {
		opts.selector = nodeSelector
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
	err = client.Call(ctx, request.Method(), request.Request(), response)
	if err != nil {
		return err
	}
	return client.Close()
}

func (r *rpcClient) stream(ctx context.Context, address string, request Request, responseChan interface{}) (Streamer, error) {
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
		return nil, errors.InternalServerError("go.micro.client", err.Error())
	}

	c, err := r.opts.transport.Dial(address, transport.WithStream())
	if err != nil {
		return nil, errors.InternalServerError("go.micro.client", fmt.Sprintf("Error sending request: %v", err))
	}

	client := newClientWithCodec(newRpcPlusCodec(msg, c, cf))
	call := client.StreamGo(request.Method(), request.Request(), responseChan)

	return &rpcStream{
		request: request,
		call:    call,
		client:  client,
	}, nil
}

func (r *rpcClient) CallRemote(ctx context.Context, address string, request Request, response interface{}) error {
	return r.call(ctx, address, request, response)
}

// TODO: Call(..., opts *Options) error {
func (r *rpcClient) Call(ctx context.Context, request Request, response interface{}) error {
	service, err := registry.GetService(request.Service())
	if err != nil {
		return errors.InternalServerError("go.micro.client", err.Error())
	}

	node, err := r.opts.selector(service)
	if err != nil {
		return err
	}

	address := node.Address
	if node.Port > 0 {
		address = fmt.Sprintf("%s:%d", address, node.Port)
	}

	return r.call(ctx, address, request, response)
}

func (r *rpcClient) StreamRemote(ctx context.Context, address string, request Request, responseChan interface{}) (Streamer, error) {
	return r.stream(ctx, address, request, responseChan)
}

func (r *rpcClient) Stream(ctx context.Context, request Request, responseChan interface{}) (Streamer, error) {
	service, err := registry.GetService(request.Service())
	if err != nil {
		return nil, errors.InternalServerError("go.micro.client", err.Error())
	}

	node, err := r.opts.selector(service)
	if err != nil {
		return nil, err
	}

	address := node.Address
	if node.Port > 0 {
		address = fmt.Sprintf("%s:%d", address, node.Port)
	}

	return r.stream(ctx, address, request, responseChan)
}

func (r *rpcClient) Publish(ctx context.Context, p Publication) error {
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
func (r *rpcClient) NewRequest(service, method string, request interface{}) Request {
	return newRpcRequest(service, method, request, r.opts.contentType)
}

func (r *rpcClient) NewProtoRequest(service, method string, request interface{}) Request {
	return newRpcRequest(service, method, request, "application/octet-stream")
}

func (r *rpcClient) NewJsonRequest(service, method string, request interface{}) Request {
	return newRpcRequest(service, method, request, "application/json")
}
