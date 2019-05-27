package server

import (
	"context"
	"sync"

	"github.com/micro/go-micro/codec"
)

// Implements the Streamer interface
type rpcStream struct {
	sync.RWMutex
	id      string
	closed  bool
	err     error
	request Request
	codec   codec.Codec
	context context.Context
}

func (r *rpcStream) Context() context.Context {
	return r.context
}

func (r *rpcStream) Request() Request {
	return r.request
}

func (r *rpcStream) Send(msg interface{}) error {
	r.Lock()
	defer r.Unlock()

	resp := codec.Message{
		Target:   r.request.Service(),
		Method:   r.request.Method(),
		Endpoint: r.request.Endpoint(),
		Id:       r.id,
		Type:     codec.Response,
	}

	return r.codec.Write(&resp, msg)
}

func (r *rpcStream) Recv(msg interface{}) error {
	r.Lock()
	defer r.Unlock()

	req := new(codec.Message)
	req.Type = codec.Request

	if err := r.codec.ReadHeader(req, req.Type); err != nil {
		// discard body
		r.codec.ReadBody(nil)
		return err
	}

	// we need to stay up to date with sequence numbers
	r.id = req.Id
	return r.codec.ReadBody(msg)
}

func (r *rpcStream) Error() error {
	r.RLock()
	defer r.RUnlock()
	return r.err
}

func (r *rpcStream) Close() error {
	r.Lock()
	defer r.Unlock()
	r.closed = true
	return r.codec.Close()
}
