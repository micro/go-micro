package grpc

import (
	"context"

	"github.com/micro/go-micro/server"
	"google.golang.org/grpc"
)

// rpcStream implements a server side Stream.
type rpcStream struct {
	s       grpc.ServerStream
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
	return r.s.Context()
}

func (r *rpcStream) Send(m interface{}) error {
	return r.s.SendMsg(m)
}

func (r *rpcStream) Recv(m interface{}) error {
	return r.s.RecvMsg(m)
}
