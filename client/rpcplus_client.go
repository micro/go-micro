// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"errors"
	"io"
	"log"
	"sync"

	"golang.org/x/net/context"
)

const (
	lastStreamResponseError = "EOS"
)

// serverError represents an error that has been returned from
// the remote side of the RPC connection.
type serverError string

func (e serverError) Error() string {
	return string(e)
}

// errShutdown holds the specific error for closing/closed connections
var errShutdown = errors.New("connection is shut down")

// call represents an active RPC.
type call struct {
	Service       string
	ServiceMethod string      // The name of the service and method to call.
	Args          interface{} // The argument to the function (*struct).
	Reply         interface{} // The reply from the function (*struct for single, chan * struct for streaming).
	Error         error       // After completion, the error status.
	Done          chan *call  // Strobes when call is complete (nil for streaming RPCs)
	Subseq        uint64      // The next expected subseq in the packets
}

// client represents an RPC client.
// There may be multiple outstanding calls associated
// with a single client, and a client may be used by
// multiple goroutines simultaneously.
type client struct {
	mutex    sync.Mutex // protects pending, seq, request
	sending  sync.Mutex
	request  request
	seq      uint64
	codec    clientCodec
	pending  map[uint64]*call
	closing  bool
	shutdown bool
}

type clientCodec interface {
	WriteRequest(*request, interface{}) error
	ReadResponseHeader(*response) error
	ReadResponseBody(interface{}) error

	Close() error
}

type request struct {
	Service       string
	ServiceMethod string   // format: "Service.Method"
	Seq           uint64   // sequence number chosen by client
	next          *request // for free list in Server
}

type response struct {
	ServiceMethod string    // echoes that of the Request
	Seq           uint64    // echoes that of the request
	Error         string    // error, if any.
	next          *response // for free list in Server
}

func (client *client) send(call *call) {
	client.sending.Lock()
	defer client.sending.Unlock()

	// Register this call.
	client.mutex.Lock()
	if client.shutdown {
		call.Error = errShutdown
		client.mutex.Unlock()
		call.done()
		return
	}
	seq := client.seq
	client.seq++
	client.pending[seq] = call
	client.mutex.Unlock()

	// Encode and send the request.
	client.request.Service = call.Service
	client.request.Seq = seq
	client.request.ServiceMethod = call.ServiceMethod
	err := client.codec.WriteRequest(&client.request, call.Args)
	if err != nil {
		client.mutex.Lock()
		call = client.pending[seq]
		delete(client.pending, seq)
		client.mutex.Unlock()
		if call != nil {
			call.Error = err
			call.done()
		}
	}
}

func (client *client) input() {
	var err error
	var resp response
	for err == nil {
		resp = response{}
		err = client.codec.ReadResponseHeader(&resp)
		if err != nil {
			if err == io.EOF && !client.closing {
				err = io.ErrUnexpectedEOF
			}
			break
		}
		seq := resp.Seq
		client.mutex.Lock()
		call := client.pending[seq]
		client.mutex.Unlock()

		switch {
		case call == nil:
			// We've got no pending call. That usually means that
			// WriteRequest partially failed, and call was already
			// removed; response is a server telling us about an
			// error reading request body. We should still attempt
			// to read error body, but there's no one to give it to.
			err = client.codec.ReadResponseBody(nil)
			if err != nil {
				err = errors.New("reading error body: " + err.Error())
			}
		case resp.Error != "":
			// We've got an error response. Give this to the request;
			// any subsequent requests will get the ReadResponseBody
			// error if there is one.
			call.Error = serverError(resp.Error)
			err = client.codec.ReadResponseBody(nil)
			if err != nil {
				err = errors.New("reading error payload: " + err.Error())
			}
			client.done(seq)
		default:
			err = client.codec.ReadResponseBody(call.Reply)
			if err != nil {
				call.Error = errors.New("reading body " + err.Error())
			}
			client.done(seq)
		}
	}
	// Terminate pending calls.
	client.sending.Lock()
	client.mutex.Lock()
	client.shutdown = true
	closing := client.closing
	for _, call := range client.pending {
		call.Error = err
		call.done()
	}
	client.mutex.Unlock()
	client.sending.Unlock()
	if err != io.EOF && !closing {
		log.Println("rpc: client protocol error:", err)
	}
}

func (client *client) done(seq uint64) {
	client.mutex.Lock()
	call := client.pending[seq]
	delete(client.pending, seq)
	client.mutex.Unlock()

	if call != nil {
		call.done()
	}
}

func (call *call) done() {
	select {
	case call.Done <- call:
		// ok
	default:
		// We don't want to block here.  It is the caller's responsibility to make
		// sure the channel has enough buffer space. See comment in Go().
		log.Println("rpc: discarding call reply due to insufficient Done chan capacity")
	}
}

func newClientWithCodec(codec clientCodec) *client {
	client := &client{
		codec:   codec,
		pending: make(map[uint64]*call),
	}
	go client.input()
	return client
}

// Close closes the client connection
func (client *client) Close() error {
	client.mutex.Lock()
	if client.shutdown || client.closing {
		client.mutex.Unlock()
		return errShutdown
	}
	client.closing = true
	client.mutex.Unlock()
	return client.codec.Close()
}

// call invokes the named function, waits for it to complete, and returns its error status.
func (client *client) Call(ctx context.Context, service string, serviceMethod string, args interface{}, reply interface{}) error {
	cal := new(call)
	cal.Service = service
	cal.ServiceMethod = serviceMethod
	cal.Args = args
	cal.Reply = reply
	cal.Done = make(chan *call, 1)
	client.send(cal)
	call := <-cal.Done
	return call.Error
}
