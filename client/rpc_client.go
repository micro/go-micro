package client

import (
	"bytes"
	"fmt"
	"sync"
	"time"

	"github.com/micro/go-micro/broker"
	"github.com/micro/go-micro/codec"
	c "github.com/micro/go-micro/context"
	"github.com/micro/go-micro/errors"
	"github.com/micro/go-micro/selector"
	"github.com/micro/go-micro/transport"

	"golang.org/x/net/context"
)

type rpcClient struct {
	once sync.Once
	opts Options
}

func newRpcClient(opt ...Option) Client {
	var once sync.Once

	opts := newOptions(opt...)

	rc := &rpcClient{
		once: once,
		opts: opts,
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

func (r *rpcClient) call(ctx context.Context, address string, req Request, resp interface{}) error {
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
		return errors.InternalServerError("go.micro.client", err.Error())
	}

	c, err := r.opts.Transport.Dial(address, transport.WithTimeout(r.opts.DialTimeout))
	if err != nil {
		return errors.InternalServerError("go.micro.client", fmt.Sprintf("Error sending request: %v", err))
	}

	var once sync.Once
	stream := &rpcStream{
		context: ctx,
		request: req,
		once:    once,
		closed:  make(chan bool),
		codec:   newRpcPlusCodec(msg, c, cf),
	}

	ch := make(chan error, 1)

	go func() {
		// defer stream close
		defer stream.Close()

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
	case err = <-ch:
	case <-time.After(r.opts.RequestTimeout):
		err = errors.New("go.micro.client", "request timeout", 408)
	}

	return err
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

	c, err := r.opts.Transport.Dial(address, transport.WithStream(), transport.WithTimeout(r.opts.DialTimeout))
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

	ch := make(chan error, 1)

	go func() {
		ch <- stream.Send(req.Request())
	}()

	select {
	case err = <-ch:
	case <-time.After(r.opts.RequestTimeout):
		err = errors.New("go.micro.client", "request timeout", 408)
	}

	return stream, err
}

func (r *rpcClient) Init(opts ...Option) error {
	for _, o := range opts {
		o(&r.opts)
	}
	return nil
}

func (r *rpcClient) Options() Options {
	return r.opts
}

func (r *rpcClient) CallRemote(ctx context.Context, address string, request Request, response interface{}, opts ...CallOption) error {
	return r.call(ctx, address, request, response)
}

func (r *rpcClient) Call(ctx context.Context, request Request, response interface{}, opts ...CallOption) error {
	var copts CallOptions
	for _, opt := range opts {
		opt(&copts)
	}

	next, err := r.opts.Selector.Select(request.Service(), copts.SelectOptions...)
	if err != nil && err == selector.ErrNotFound {
		return errors.NotFound("go.micro.client", err.Error())
	} else if err != nil {
		return errors.InternalServerError("go.micro.client", err.Error())
	}

	var grr error

	for i := 0; i < r.opts.Retries; i++ {
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

		grr = r.call(ctx, address, request, response)
		r.opts.Selector.Mark(request.Service(), node, grr)

		// if the call succeeded lets bail early
		if grr == nil {
			return nil
		}
	}

	return grr
}

func (r *rpcClient) StreamRemote(ctx context.Context, address string, request Request, opts ...CallOption) (Streamer, error) {
	return r.stream(ctx, address, request)
}

func (r *rpcClient) Stream(ctx context.Context, request Request, opts ...CallOption) (Streamer, error) {
	var copts CallOptions
	for _, opt := range opts {
		opt(&copts)
	}

	next, err := r.opts.Selector.Select(request.Service(), copts.SelectOptions...)
	if err != nil && err == selector.ErrNotFound {
		return nil, errors.NotFound("go.micro.client", err.Error())
	} else if err != nil {
		return nil, errors.InternalServerError("go.micro.client", err.Error())
	}

	var stream Streamer
	var grr error

	for i := 0; i < r.opts.Retries; i++ {
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

		stream, grr = r.stream(ctx, address, request)
		r.opts.Selector.Mark(request.Service(), node, grr)

		// bail early if succeeds
		if grr == nil {
			return stream, nil
		}
	}

	return stream, grr
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
