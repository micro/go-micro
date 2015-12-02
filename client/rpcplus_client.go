// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"errors"
	"io"
	"log"
	"reflect"
	"sync"

	"github.com/youtube/vitess/go/trace"
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
	ServiceMethod string      // The name of the service and method to call.
	Args          interface{} // The argument to the function (*struct).
	Reply         interface{} // The reply from the function (*struct for single, chan * struct for streaming).
	Error         error       // After completion, the error status.
	Done          chan *call  // Strobes when call is complete (nil for streaming RPCs)
	Stream        bool        // True for a streaming RPC call, false otherwise
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
			if !(call.Stream && resp.Error == lastStreamResponseError) {
				call.Error = serverError(resp.Error)
			}
			err = client.codec.ReadResponseBody(nil)
			if err != nil {
				err = errors.New("reading error payload: " + err.Error())
			}
			client.done(seq)
		case call.Stream:
			// call.Reply is a chan *T2
			// we need to create a T2 and get a *T2 back
			value := reflect.New(reflect.TypeOf(call.Reply).Elem().Elem()).Interface()
			err = client.codec.ReadResponseBody(value)
			if err != nil {
				call.Error = errors.New("reading body " + err.Error())
			} else {
				// writing on the channel could block forever. For
				// instance, if a client calls 'close', this might block
				// forever.  the current suggestion is for the
				// client to drain the receiving channel in that case
				reflect.ValueOf(call.Reply).Send(reflect.ValueOf(value))
			}
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
	if call.Stream {
		// need to close the channel. client won't be able to read any more.
		reflect.ValueOf(call.Reply).Close()
		return
	}

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

// Go invokes the function asynchronously.  It returns the call structure representing
// the invocation.  The done channel will signal when the call is complete by returning
// the same call object.  If done is nil, Go will allocate a new channel.
// If non-nil, done must be buffered or Go will deliberately crash.
func (client *client) Go(ctx context.Context, serviceMethod string, args interface{}, reply interface{}, done chan *call) *call {
	span := trace.NewSpanFromContext(ctx)
	span.StartClient(serviceMethod)
	defer span.Finish()

	cal := new(call)
	cal.ServiceMethod = serviceMethod
	cal.Args = args
	cal.Reply = reply
	if done == nil {
		done = make(chan *call, 10) // buffered.
	} else {
		// If caller passes done != nil, it must arrange that
		// done has enough buffer for the number of simultaneous
		// RPCs that will be using that channel.  If the channel
		// is totally unbuffered, it's best not to run at all.
		if cap(done) == 0 {
			log.Panic("rpc: done channel is unbuffered")
		}
	}
	cal.Done = done
	client.send(cal)
	return cal
}

// StreamGo invokes the streaming function asynchronously.  It returns the call structure representing
// the invocation.
func (client *client) StreamGo(serviceMethod string, args interface{}, replyStream interface{}) *call {
	// first check the replyStream object is a stream of pointers to a data structure
	typ := reflect.TypeOf(replyStream)
	// FIXME: check the direction of the channel, maybe?
	if typ.Kind() != reflect.Chan || typ.Elem().Kind() != reflect.Ptr {
		log.Panic("rpc: replyStream is not a channel of pointers")
		return nil
	}

	call := new(call)
	call.ServiceMethod = serviceMethod
	call.Args = args
	call.Reply = replyStream
	call.Stream = true
	call.Subseq = 0
	client.send(call)
	return call
}

// call invokes the named function, waits for it to complete, and returns its error status.
func (client *client) Call(ctx context.Context, serviceMethod string, args interface{}, reply interface{}) error {
	call := <-client.Go(ctx, serviceMethod, args, reply, make(chan *call, 1)).Done
	return call.Error
}
