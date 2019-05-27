package client

import (
	"context"
	"io"
	"sync"

	"github.com/micro/go-micro/codec"
)

// Implements the streamer interface
type rpcStream struct {
	sync.RWMutex
	id       string
	closed   chan bool
	err      error
	request  Request
	response Response
	codec    codec.Codec
	context  context.Context
}

func (r *rpcStream) isClosed() bool {
	select {
	case <-r.closed:
		return true
	default:
		return false
	}
}

func (r *rpcStream) Context() context.Context {
	return r.context
}

func (r *rpcStream) Request() Request {
	return r.request
}

func (r *rpcStream) Response() Response {
	return r.response
}

func (r *rpcStream) Send(msg interface{}) error {
	r.Lock()
	defer r.Unlock()

	if r.isClosed() {
		r.err = errShutdown
		return errShutdown
	}

	req := codec.Message{
		Id:       r.id,
		Target:   r.request.Service(),
		Method:   r.request.Method(),
		Endpoint: r.request.Endpoint(),
		Type:     codec.Request,
	}

	if err := r.codec.Write(&req, msg); err != nil {
		r.err = err
		return err
	}

	return nil
}

func (r *rpcStream) Recv(msg interface{}) error {
	r.Lock()
	defer r.Unlock()

	if r.isClosed() {
		r.err = errShutdown
		return errShutdown
	}

	var resp codec.Message

	if err := r.codec.ReadHeader(&resp, codec.Response); err != nil {
		if err == io.EOF && !r.isClosed() {
			r.err = io.ErrUnexpectedEOF
			return io.ErrUnexpectedEOF
		}
		r.err = err
		return err
	}

	switch {
	case len(resp.Error) > 0:
		// We've got an error response. Give this to the request;
		// any subsequent requests will get the ReadResponseBody
		// error if there is one.
		if resp.Error != lastStreamResponseError {
			r.err = serverError(resp.Error)
		} else {
			r.err = io.EOF
		}
		if err := r.codec.ReadBody(nil); err != nil {
			r.err = err
		}
	default:
		if err := r.codec.ReadBody(msg); err != nil {
			r.err = err
		}
	}

	return r.err
}

func (r *rpcStream) Error() error {
	r.RLock()
	defer r.RUnlock()
	return r.err
}

func (r *rpcStream) Close() error {
	select {
	case <-r.closed:
		return nil
	default:
		close(r.closed)
		return r.codec.Close()
	}
}
