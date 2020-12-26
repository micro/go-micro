// Package json provides a json codec
package json

import (
	stdjson "encoding/json"
	"io"

	"github.com/golang/protobuf/proto"
	"github.com/micro/go-micro/v2/codec"
	segjson "github.com/segmentio/encoding/json"
)

type Codec struct {
	Conn       io.ReadWriteCloser
	StdDecoder *stdjson.Decoder
	SegEncoder *segjson.Encoder
	SegDecoder *segjson.Decoder
}

func (c *Codec) ReadHeader(m *codec.Message, t codec.MessageType) error {
	return nil
}

func (c *Codec) ReadBody(b interface{}) error {
	if b == nil {
		return nil
	}
	if pb, ok := b.(proto.Message); ok {
		return jsonpbUnmarshaler.UnmarshalNext(c.StdDecoder, pb)
	}
	return c.SegDecoder.Decode(b)
}

func (c *Codec) Write(m *codec.Message, b interface{}) error {
	if b == nil {
		return nil
	}
	return c.SegEncoder.Encode(b)
}

func (c *Codec) Close() error {
	return c.Conn.Close()
}

func (c *Codec) String() string {
	return "json"
}

func NewCodec(conn io.ReadWriteCloser) codec.Codec {
	c := &Codec{
		Conn:       conn,
		SegDecoder: segjson.NewDecoder(conn),
		SegEncoder: segjson.NewEncoder(conn),
		StdDecoder: stdjson.NewDecoder(conn),
	}
	c.SegEncoder.SetEscapeHTML(false)
	c.SegEncoder.SetSortMapKeys(false)
	c.SegDecoder.ZeroCopy()

	return c
}
