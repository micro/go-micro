package msgpackrpc

import (
	"errors"
	"fmt"
	"io"

	"github.com/micro/go-micro/v2/codec"
	"github.com/tinylib/msgp/msgp"
)

type msgpackCodec struct {
	rwc  io.ReadWriteCloser
	mt   codec.MessageType
	body bool
}

func (c *msgpackCodec) Close() error {
	return c.rwc.Close()
}

func (c *msgpackCodec) String() string {
	return "msgpack-rpc"
}

// ReadHeader reads the header from the wire.
func (c *msgpackCodec) ReadHeader(m *codec.Message, mt codec.MessageType) error {
	c.mt = mt

	switch mt {
	case codec.Request:
		var h Request

		if err := msgp.Decode(c.rwc, &h); err != nil {
			return err
		}

		c.body = h.hasBody
		m.Id = h.ID
		m.Endpoint = h.Method

	case codec.Response:
		var h Response

		if err := msgp.Decode(c.rwc, &h); err != nil {
			return err
		}

		c.body = h.hasBody
		m.Id = h.ID
		m.Error = h.Error

	case codec.Event:
		var h Notification

		if err := msgp.Decode(c.rwc, &h); err != nil {
			return err
		}

		c.body = h.hasBody
		m.Endpoint = h.Method

	default:
		return errors.New("Unrecognized message type")
	}

	return nil
}

// ReadBody reads the body of the message. It is assumed the value being
// decoded into is a satisfies the msgp.Decodable interface.
func (c *msgpackCodec) ReadBody(v interface{}) error {
	if !c.body {
		return nil
	}

	r := msgp.NewReader(c.rwc)

	// Body is present, but no value to decode into.
	if v == nil {
		return r.Skip()
	}

	switch c.mt {
	case codec.Request, codec.Response, codec.Event:
		return decodeBody(r, v)
	default:
		return fmt.Errorf("Unrecognized message type: %v", c.mt)
	}
}

// Write writes a message to the wire which contains the header followed by the body.
// The body is assumed to satisfy the msgp.Encodable interface.
func (c *msgpackCodec) Write(m *codec.Message, b interface{}) error {
	switch m.Type {
	case codec.Request:
		h := Request{
			ID:     m.Id,
			Method: m.Endpoint,
			Body:   b,
		}

		return msgp.Encode(c.rwc, &h)

	case codec.Response:
		h := Response{
			ID:   m.Id,
			Body: b,
		}

		h.Error = m.Error

		return msgp.Encode(c.rwc, &h)

	case codec.Event:
		h := Notification{
			Method: m.Endpoint,
			Body:   b,
		}

		return msgp.Encode(c.rwc, &h)

	default:
		return fmt.Errorf("Unrecognized message type: %v", m.Type)
	}
}

func NewCodec(rwc io.ReadWriteCloser) codec.Codec {
	return &msgpackCodec{
		rwc: rwc,
	}
}
