// Package proto provides a proto codec
package proto

import (
	"io"
	"io/ioutil"

	"github.com/asim/go-micro/v3/codec"
	"github.com/golang/protobuf/proto"
)

type Codec struct {
	Conn io.ReadWriteCloser
}

func (c *Codec) ReadHeader(m *codec.Message, t codec.MessageType) error {
	return nil
}

func (c *Codec) ReadBody(b interface{}) error {
	if b == nil {
		return nil
	}
	buf, err := ioutil.ReadAll(c.Conn)
	if err != nil {
		return err
	}
	m, ok := b.(proto.Message)
	if !ok {
		return codec.ErrInvalidMessage
	}
	return proto.Unmarshal(buf, m)
}

func (c *Codec) Write(m *codec.Message, b interface{}) error {
	if b == nil {
		// Nothing to write
		return nil
	}
	p, ok := b.(proto.Message)
	if !ok {
		return codec.ErrInvalidMessage
	}
	buf, err := proto.Marshal(p)
	if err != nil {
		return err
	}
	_, err = c.Conn.Write(buf)
	return err
}

func (c *Codec) Close() error {
	return c.Conn.Close()
}

func (c *Codec) String() string {
	return "proto"
}

func NewCodec(c io.ReadWriteCloser) codec.Codec {
	return &Codec{
		Conn: c,
	}
}
