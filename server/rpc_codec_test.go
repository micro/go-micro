package server

import (
	"bytes"
	"errors"
	"testing"

	"github.com/micro/go-micro/codec"
	"github.com/micro/go-micro/transport"
)

// testCodec is a dummy codec that only knows how to encode nil bodies
type testCodec struct {
	buf *bytes.Buffer
}

type testSocket struct {
}

// TestCodecWriteError simulates what happens when a codec is unable
// to encode a message (e.g. a missing branch of an "oneof" message in
// protobufs)
//
// We expect an error to be sent to the socket. Previously the socket
// would remain open with no bytes sent, leading to client-side
// timeouts.
func TestCodecWriteError(t *testing.T) {
	socket := testSocket{}
	message := transport.Message{
		Header: map[string]string{},
		Body:   []byte{},
	}

	rwc := readWriteCloser{
		rbuf: new(bytes.Buffer),
		wbuf: new(bytes.Buffer),
	}

	c := rpcPlusCodec{
		buf: &rwc,
		codec: &testCodec{
			buf: rwc.wbuf,
		},
		req:    &message,
		socket: socket,
	}

	err := c.WriteResponse(&response{
		ServiceMethod: "Service.Method",
		Seq:           0,
		Error:         "",
		next:          nil,
	}, "body", false)

	if err != nil {
		t.Fatalf(`Expected WriteResponse to fail; got "%+v" instead`, err)
	}

	const expectedError = "Unable to encode body: simulating a codec write failure"
	actualError := rwc.wbuf.String()
	if actualError != expectedError {
		t.Fatalf(`Expected error "%+v" in the write buffer, got "%+v" instead`, expectedError, actualError)
	}
}

func (c *testCodec) ReadHeader(message *codec.Message, typ codec.MessageType) error {
	return nil
}

func (c *testCodec) ReadBody(dest interface{}) error {
	return nil
}

func (c *testCodec) Write(message *codec.Message, dest interface{}) error {
	if dest != nil {
		return errors.New("simulating a codec write failure")
	}
	c.buf.Write([]byte(message.Error))
	return nil
}

func (c *testCodec) Close() error {
	return nil
}

func (c *testCodec) String() string {
	return "string"
}

func (s testSocket) Recv(message *transport.Message) error {
	return nil
}

func (s testSocket) Send(message *transport.Message) error {
	return nil
}

func (s testSocket) Close() error {
	return nil
}
