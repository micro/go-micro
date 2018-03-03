package server

import (
	"context"
	"sync"
)

// Implements the Streamer interface
type rpcStream struct {
	sync.RWMutex
	seq     uint64
	closed  bool
	err     error
	request Request
	codec   serverCodec
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

	resp := response{
		ServiceMethod: r.request.Method(),
		Seq:           r.seq,
	}

	return r.codec.WriteResponse(&resp, msg, false)
}

func (r *rpcStream) Recv(msg interface{}) error {
	r.Lock()
	defer r.Unlock()

	req := request{}

	if err := r.codec.ReadRequestHeader(&req, false); err != nil {
		// discard body
		r.codec.ReadRequestBody(nil)
		return err
	}

	// we need to stay up to date with sequence numbers
	r.seq = req.Seq
	return r.codec.ReadRequestBody(msg)
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
