package client

import (
	"golang.org/x/net/context"

	"github.com/piemapping/plugged/trace"
)

type Client interface {
	NewPublication(topic string, msg interface{}) Publication
	NewRequest(service, method string, req interface{}) Request
	NewProtoRequest(service, method string, req interface{}) Request
	NewJsonRequest(service, method string, req interface{}) Request
	Call(ctx context.Context, req Request, rsp interface{}) error
	CallRemote(ctx context.Context, addr string, req Request, rsp interface{}) error
	Stream(ctx context.Context, req Request, rspChan interface{}) (Streamer, error)
	StreamRemote(ctx context.Context, addr string, req Request, rspChan interface{}) (Streamer, error)
	Publish(ctx context.Context, p Publication) error
}

type Publication interface {
	Topic() string
	Message() interface{}
	ContentType() string
}

type Request interface {
	Service() string
	Method() string
	ContentType() string
	Request() interface{}
}

type Streamer interface {
	Request() Request
	Error() error
	Close() error
}

type Option func(*options)

var (
	DefaultClient Client = newRpcClient()
)

func Call(ctx context.Context, request Request, response interface{}) error {
	// Create a trace just before sending the message over the wire
	tr, ctx := createTrace(ctx, trace.SenderSend, request.Service(), request.Method(), request.Request(), nil)
	// and submit it...
	submitTrace(tr)

	err := DefaultClient.Call(ctx, request, response)

	// Create a trace just after getting the response (or error) from remote server
	tr, ctx = createTrace(ctx, trace.SenderReceive, request.Service(), request.Method(), response, err)
	// and submit it...
	submitTrace(tr)

	return err
}

func CallRemote(ctx context.Context, address string, request Request, response interface{}) error {
	return DefaultClient.CallRemote(ctx, address, request, response)
}

func Stream(ctx context.Context, request Request, responseChan interface{}) (Streamer, error) {
	return DefaultClient.Stream(ctx, request, responseChan)
}

func StreamRemote(ctx context.Context, address string, request Request, responseChan interface{}) (Streamer, error) {
	return DefaultClient.StreamRemote(ctx, address, request, responseChan)
}

func Publish(ctx context.Context, p Publication) error {
	return DefaultClient.Publish(ctx, p)
}

func NewClient(opt ...Option) Client {
	return newRpcClient(opt...)
}

func NewPublication(topic string, message interface{}) Publication {
	return DefaultClient.NewPublication(topic, message)
}

func NewRequest(service, method string, request interface{}) Request {
	return DefaultClient.NewRequest(service, method, request)
}

func NewProtoRequest(service, method string, request interface{}) Request {
	return DefaultClient.NewProtoRequest(service, method, request)
}

func NewJsonRequest(service, method string, request interface{}) Request {
	return DefaultClient.NewJsonRequest(service, method, request)
}
