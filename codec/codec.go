// Package codec is an interface for encoding messages
package codec

import (
	"io"
)

const (
	Error MessageType = iota
	Request
	Response
	Publication
)

type MessageType int

// Takes in a connection/buffer and returns a new Codec
type NewCodec func(io.ReadWriteCloser) Codec

// Codec encodes/decodes various types of messages used within go-micro.
// ReadHeader and ReadBody are called in pairs to read requests/responses
// from the connection. Close is called when finished with the
// connection. ReadBody may be called with a nil argument to force the
// body to be read and discarded.
type Codec interface {
	ReadHeader(*Message, MessageType) error
	ReadBody(interface{}) error
	Write(*Message, interface{}) error
	Close() error
	String() string
}

// Message represents detailed information about
// the communication, likely followed by the body.
// In the case of an error, body may be nil.
type Message struct {
	Id     uint64
	Type   MessageType
	Target string
	Method string
	Error  string
	Header map[string]string
}
