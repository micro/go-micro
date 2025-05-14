package grpc

import (
	"context"
	"io"
	"sync"

	"go-micro.dev/v5/client"
	"google.golang.org/grpc"
)

// Implements the streamer interface.
type grpcStream struct {
	sync.RWMutex
	closed   bool
	err      error
	stream   grpc.ClientStream
	request  client.Request
	response client.Response
	context  context.Context
	cancel   func()
	release  func(error)
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
	if err = g.stream.RecvMsg(msg); err != nil {
		if err != io.EOF {
			g.setError(err)
		}
		return err
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

func (g *grpcStream) CloseSend() error {
	return g.stream.CloseSend()
}

func (g *grpcStream) Close() error {
	g.Lock()
	defer g.Unlock()

	if g.closed {
		return nil
	}
	// cancel the context
	g.cancel()
	g.closed = true
	// release back to pool
	g.release(g.err)
	return nil
}
