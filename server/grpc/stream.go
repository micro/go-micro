package grpc

import (
	"context"

	"github.com/micro/go-micro/v3/server"
	"google.golang.org/grpc"
)

// rpcStream implements a server side Stream.
type rpcStream struct {
	// embed the grpc stream so we can access it
	grpc.ServerStream

	request server.Request
}

func (r *rpcStream) Close() error {
	return nil
}

func (r *rpcStream) Error() error {
	return nil
}

func (r *rpcStream) Request() server.Request {
	return r.request
}

func (r *rpcStream) Context() context.Context {
	return r.ServerStream.Context()
}

func (r *rpcStream) Send(m interface{}) error {
	return r.ServerStream.SendMsg(m)
}

func (r *rpcStream) Recv(m interface{}) error {
	return r.ServerStream.RecvMsg(m)
}
