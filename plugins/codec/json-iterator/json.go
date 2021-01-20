// Package json iterator provides a json codec
package json

import (
	//	jsonstd "encoding/json"
	"io"

	jsoniter "github.com/json-iterator/go"
	"github.com/asim/go-micro/v3/codec"
)

type Codec struct {
	Conn io.ReadWriteCloser

	//	StdEncoder  *jsonstd.Encoder
	//	StdDecoder  *jsonstd.Decoder
	IterEncoder *jsoniter.Encoder
	IterDecoder *jsoniter.Decoder
}

func (c *Codec) ReadHeader(m *codec.Message, t codec.MessageType) error {
	return nil
}

func (c *Codec) ReadBody(b interface{}) error {
	if b == nil {
		return nil
	}
	//	if pb, ok := b.(proto.Message); ok {
	//		return jsonpb.UnmarshalNext(c.StdDecoder, pb)
	//	}
	return c.IterDecoder.Decode(b)
}

func (c *Codec) Write(m *codec.Message, b interface{}) error {
	if b == nil {
		return nil
	}
	return c.IterEncoder.Encode(b)
}

func (c *Codec) Close() error {
	return c.Conn.Close()
}

func (c *Codec) String() string {
	return "json"
}

func NewCodec(c io.ReadWriteCloser) codec.Codec {
	return &Codec{
		Conn: c,
		//	StdDecoder:  jsonstd.NewDecoder(c),
		//	StdEncoder:  jsonstd.NewEncoder(c),
		IterDecoder: jsoniter.NewDecoder(c),
		IterEncoder: jsoniter.NewEncoder(c),
	}
}
