package client

import (
	"bytes"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	c "github.com/myodc/go-micro/context"
	"github.com/myodc/go-micro/errors"
	"github.com/myodc/go-micro/registry"
	"github.com/myodc/go-micro/transport"

	rpc "github.com/youtube/vitess/go/rpcplus"
	js "github.com/youtube/vitess/go/rpcplus/jsonrpc"
	pb "github.com/youtube/vitess/go/rpcplus/pbrpc"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type headerRoundTripper struct {
	r http.RoundTripper
}

type rpcClient struct {
	opts options
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func newRpcClient(opt ...Option) Client {
	var opts options

	for _, o := range opt {
		o(&opts)
	}

	if opts.transport == nil {
		opts.transport = transport.DefaultTransport
	}

	return &rpcClient{
		opts: opts,
	}
}

func (t *headerRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Set("X-Client-Version", "1.0")
	return t.r.RoundTrip(r)
}

func (r *rpcClient) call(ctx context.Context, address string, request Request, response interface{}) error {
	switch request.ContentType() {
	case "application/grpc":
		cc, err := grpc.Dial(address)
		if err != nil {
			return errors.InternalServerError("go.micro.client", fmt.Sprintf("Error connecting to server: %v", err))
		}
		if err := grpc.Invoke(ctx, request.Method(), request.Request(), response, cc); err != nil {
			return errors.InternalServerError("go.micro.client", fmt.Sprintf("Error sending request: %v", err))
		}
		return nil
	}

	pReq := &rpc.Request{
		ServiceMethod: request.Method(),
	}

	reqB := bytes.NewBuffer(nil)
	defer reqB.Reset()
	buf := &buffer{
		reqB,
	}

	var cc rpc.ClientCodec
	switch request.ContentType() {
	case "application/octet-stream":
		cc = pb.NewClientCodec(buf)
	case "application/json":
		cc = js.NewClientCodec(buf)
	default:
		return errors.InternalServerError("go.micro.client", fmt.Sprintf("Unsupported request type: %s", request.ContentType()))
	}

	err := cc.WriteRequest(pReq, request.Request())
	if err != nil {
		return errors.InternalServerError("go.micro.client", fmt.Sprintf("Error writing request: %v", err))
	}

	msg := &transport.Message{
		Header: make(map[string]string),
		Body:   reqB.Bytes(),
	}

	md, ok := c.GetMetadata(ctx)
	if ok {
		for k, v := range md {
			msg.Header[k] = v
		}
	}

	msg.Header["Content-Type"] = request.ContentType()

	c, err := r.opts.transport.Dial(address)
	if err != nil {
		return errors.InternalServerError("go.micro.client", fmt.Sprintf("Error sending request: %v", err))
	}

	rsp, err := c.Send(msg)
	if err != nil {
		return errors.InternalServerError("go.micro.client", fmt.Sprintf("Error sending request: %v", err))
	}

	rspB := bytes.NewBuffer(rsp.Body)
	defer rspB.Reset()
	rBuf := &buffer{
		rspB,
	}

	switch rsp.Header["Content-Type"] {
	case "application/octet-stream":
		cc = pb.NewClientCodec(rBuf)
	case "application/json":
		cc = js.NewClientCodec(rBuf)
	default:
		return errors.InternalServerError("go.micro.client", string(rsp.Body))
	}

	pRsp := &rpc.Response{}
	err = cc.ReadResponseHeader(pRsp)
	if err != nil {
		return errors.InternalServerError("go.micro.client", fmt.Sprintf("Error reading response headers: %v", err))
	}

	if len(pRsp.Error) > 0 {
		return errors.Parse(pRsp.Error)
	}

	err = cc.ReadResponseBody(response)
	if err != nil {
		return errors.InternalServerError("go.micro.client", fmt.Sprintf("Error reading response body: %v", err))
	}

	return nil
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

	if len(service.Nodes) == 0 {
		return errors.NotFound("go.micro.client", "Service not found")
	}

	n := rand.Int() % len(service.Nodes)
	node := service.Nodes[n]

	address := node.Address
	if node.Port > 0 {
		address = fmt.Sprintf("%s:%d", address, node.Port)
	}

	return r.call(ctx, address, request, response)
}

func (r *rpcClient) NewRequest(service, method string, request interface{}) Request {
	return r.NewProtoRequest(service, method, request)
}

func (r *rpcClient) NewProtoRequest(service, method string, request interface{}) Request {
	return newRpcRequest(service, method, request, "application/octet-stream")
}

func (r *rpcClient) NewJsonRequest(service, method string, request interface{}) Request {
	return newRpcRequest(service, method, request, "application/json")
}
