package grpc

import (
	"context"
	"io"
	"sync"

	"github.com/micro/go-micro/client"
	"google.golang.org/grpc"
)

// Implements the streamer interface
type grpcStream struct {
	sync.RWMutex
	err     error
	conn    *grpc.ClientConn
	request client.Request
	stream  grpc.ClientStream
	context context.Context
}

func (g *grpcStream) Context() context.Context {
	return g.context
}

func (g *grpcStream) Request() client.Request {
	return g.request
}

func (g *grpcStream) Response() client.Response {
	return nil
}

func (g *grpcStream) Send(msg interface{}) error {
	if err := g.stream.SendMsg(msg); err != nil {
		g.setError(err)
		return err
	}
	return nil
}

func (g *grpcStream) Recv(msg interface{}) (err error) {
	defer g.setError(err)
	if err = g.stream.RecvMsg(msg); err != nil {
		if err == io.EOF {
			// #202 - inconsistent gRPC stream behavior
			// the only way to tell if the stream is done is when we get a EOF on the Recv
			// here we should close the underlying gRPC ClientConn
			closeErr := g.conn.Close()
			if closeErr != nil {
				err = closeErr
			}
		}
	}
	return
}

func (g *grpcStream) Error() error {
	g.RLock()
	defer g.RUnlock()
	return g.err
}

func (g *grpcStream) setError(e error) {
	g.Lock()
	g.err = e
	g.Unlock()
}

// Close the gRPC send stream
// #202 - inconsistent gRPC stream behavior
// The underlying gRPC stream should not be closed here since the
// stream should still be able to receive after this function call
// TODO: should the conn be closed in another way?
func (g *grpcStream) Close() error {
	return g.stream.CloseSend()
}
