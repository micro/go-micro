package grpc

import (
	"context"
	"io"
	"sync"

	"github.com/micro/go-micro/v2/client"
	"google.golang.org/grpc"
)

// Implements the streamer interface
type grpcStream struct {
	sync.RWMutex
	closed   bool
	err      error
	conn     *grpc.ClientConn
	stream   grpc.ClientStream
	request  client.Request
	response client.Response
	context  context.Context
	cancel   func()
}

func (g *grpcStream) Context() context.Context {
	return g.context
}

func (g *grpcStream) Request() client.Request {
	return g.request
}

func (g *grpcStream) Response() client.Response {
	return g.response
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
		// #202 - inconsistent gRPC stream behavior
		// the only way to tell if the stream is done is when we get a EOF on the Recv
		// here we should close the underlying gRPC ClientConn
		closeErr := g.Close()
		if err == io.EOF && closeErr != nil {
			err = closeErr
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
	g.Lock()
	defer g.Unlock()

	if g.closed {
		return nil
	}
	// cancel the context
	g.cancel()
	g.closed = true
	g.stream.CloseSend()
	return g.conn.Close()
}
