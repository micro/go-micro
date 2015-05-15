package client

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"time"

	"github.com/myodc/go-micro/errors"
	"github.com/myodc/go-micro/registry"
	rpc "github.com/youtube/vitess/go/rpcplus"
	js "github.com/youtube/vitess/go/rpcplus/jsonrpc"
	pb "github.com/youtube/vitess/go/rpcplus/pbrpc"
	ctx "golang.org/x/net/context"
	"google.golang.org/grpc"
)

type headerRoundTripper struct {
	r http.RoundTripper
}

type RpcClient struct{}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func (t *headerRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Set("X-Client-Version", "1.0")
	return t.r.RoundTrip(r)
}

func (r *RpcClient) call(address, path string, request Request, response interface{}) error {
	switch request.ContentType() {
	case "application/grpc":
		cc, err := grpc.Dial(address)
		if err != nil {
			return errors.InternalServerError("go.micro.client", fmt.Sprintf("Error connecting to server: %v", err))
		}
		if err := grpc.Invoke(ctx.Background(), path, request.Request(), response, cc); err != nil {
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

	client := &http.Client{}
	client.Transport = &headerRoundTripper{http.DefaultTransport}

	request.Headers().Set("Content-Type", request.ContentType())

	hreq := &http.Request{
		Method: "POST",
		URL: &url.URL{
			Scheme: "http",
			Host:   address,
			Path:   path,
		},
		Header:        request.Headers().(http.Header),
		Body:          buf,
		ContentLength: int64(reqB.Len()),
		Host:          address,
	}

	rsp, err := client.Do(hreq)
	if err != nil {
		return errors.InternalServerError("go.micro.client", fmt.Sprintf("Error sending request: %v", err))
	}
	defer rsp.Body.Close()

	b, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		return errors.InternalServerError("go.micro.client", fmt.Sprintf("Error reading response: %v", err))
	}

	rspB := bytes.NewBuffer(b)
	defer rspB.Reset()
	rBuf := &buffer{
		rspB,
	}

	switch rsp.Header.Get("Content-Type") {
	case "application/octet-stream":
		cc = pb.NewClientCodec(rBuf)
	case "application/json":
		cc = js.NewClientCodec(rBuf)
	default:
		return errors.InternalServerError("go.micro.client", string(b))
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

func (r *RpcClient) CallRemote(address, path string, request Request, response interface{}) error {
	return r.call(address, path, request, response)
}

// TODO: Call(..., opts *Options) error {
func (r *RpcClient) Call(request Request, response interface{}) error {
	service, err := registry.GetService(request.Service())
	if err != nil {
		return errors.InternalServerError("go.micro.client", err.Error())
	}

	if len(service.Nodes()) == 0 {
		return errors.NotFound("go.micro.client", "Service not found")
	}

	n := rand.Int() % len(service.Nodes())
	node := service.Nodes()[n]
	address := fmt.Sprintf("%s:%d", node.Address(), node.Port())
	return r.call(address, "/_rpc", request, response)
}

func (r *RpcClient) NewRequest(service, method string, request interface{}) *RpcRequest {
	return r.NewProtoRequest(service, method, request)
}

func (r *RpcClient) NewProtoRequest(service, method string, request interface{}) *RpcRequest {
	return newRpcRequest(service, method, request, "application/octet-stream")
}

func (r *RpcClient) NewJsonRequest(service, method string, request interface{}) *RpcRequest {
	return newRpcRequest(service, method, request, "application/json")
}

func NewRpcClient() *RpcClient {
	return &RpcClient{}
}
