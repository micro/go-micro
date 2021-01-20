// Package bsonrpc provides a bson-rpc codec
package bsonrpc

import (
	"bytes"
	"fmt"
	"io"

	"github.com/asim/go-bson"
	"github.com/micro/go-micro/v2/codec"
)

type bsonCodec struct {
	buf *bytes.Buffer
	mt  codec.MessageType
	rwc io.ReadWriteCloser

	c *clientCodec
	s *serverCodec
}

func (b *bsonCodec) Close() error {
	b.buf.Reset()
	return b.rwc.Close()
}

func (b *bsonCodec) String() string {
	return "bson-rpc"
}

func (b *bsonCodec) Write(m *codec.Message, body interface{}) error {
	switch m.Type {
	case codec.Request:
		return b.c.Write(m, body)
	case codec.Response:
		return b.s.Write(m, body)
	case codec.Event:
		data, err := bson.Marshal(body)
		if err != nil {
			return err
		}
		_, err = b.rwc.Write(data)
		return err
	default:
		return fmt.Errorf("Unrecognised message type: %v", m.Type)
	}
}

func (b *bsonCodec) ReadHeader(m *codec.Message, mt codec.MessageType) error {
	b.buf.Reset()
	b.mt = mt

	switch mt {
	case codec.Request:
		return b.s.ReadHeader(m)
	case codec.Response:
		return b.c.ReadHeader(m)
	case codec.Event:
		io.Copy(b.buf, b.rwc)
	default:
		return fmt.Errorf("Unrecognised message type: %v", mt)
	}
	return nil
}

func (b *bsonCodec) ReadBody(body interface{}) error {
	switch b.mt {
	case codec.Request:
		return b.s.ReadBody(body)
	case codec.Response:
		return b.c.ReadBody(body)
	case codec.Event:
		if body != nil {
			return bson.Unmarshal(b.buf.Bytes(), body)
		}
	default:
		return fmt.Errorf("Unrecognised message type: %v", b.mt)
	}
	return nil
}

func NewCodec(rwc io.ReadWriteCloser) codec.Codec {
	return &bsonCodec{
		buf: bytes.NewBuffer(nil),
		rwc: rwc,
		c:   newClientCodec(rwc),
		s:   newServerCodec(rwc),
	}
}
