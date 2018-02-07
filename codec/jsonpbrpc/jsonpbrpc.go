package jsonpbrpc

import (
	"bytes"
	"fmt"
	"io"
	"sync"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/micro/go-micro/codec"
)

type flusher interface {
	Flush() error
}

type protoCodec struct {
	sync.Mutex
	rwc  io.ReadWriteCloser
	mt   codec.MessageType
	buf  *bytes.Buffer
	mrs  *jsonpb.Marshaler
	umrs *jsonpb.Unmarshaler
}

func (c *protoCodec) Close() error {
	c.buf.Reset()
	return c.rwc.Close()
}

func (c *protoCodec) String() string {
	return "jsonpb-rpc"
}

func (c *protoCodec) Write(m *codec.Message, b interface{}) error {
	switch m.Type {
	case codec.Request:
		c.Lock()
		defer c.Unlock()
		// This is protobuf, of course we copy it.
		pbr := &Request{ServiceMethod: &m.Method, Seq: &m.Id}
		err := c.mrs.Marshal(c.rwc, pbr)
		if err != nil {
			return err
		}
		// Of course this is a protobuf! Trust me or detonate the program.
		err = c.mrs.Marshal(c.rwc, b.(proto.Message))
		if err != nil {
			return err
		}
		if flusher, ok := c.rwc.(flusher); ok {
			err = flusher.Flush()
		}
	case codec.Response:
		c.Lock()
		defer c.Unlock()
		rtmp := &Response{ServiceMethod: &m.Method, Seq: &m.Id, Error: &m.Error}
		err := c.mrs.Marshal(c.rwc, rtmp)
		if err != nil {
			return err
		}
		if pb, ok := b.(proto.Message); ok {
			err = c.mrs.Marshal(c.rwc, pb)
			if err != nil {
				return err
			}
		}
		if flusher, ok := c.rwc.(flusher); ok {
			err = flusher.Flush()
		}
	case codec.Publication:
		err := c.mrs.Marshal(c.rwc, b.(proto.Message))
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("Unrecognised message type: %v", m.Type)
	}
	return nil
}

func (c *protoCodec) ReadHeader(m *codec.Message, mt codec.MessageType) error {
	c.buf.Reset()
	c.mt = mt

	switch mt {
	case codec.Request:
		rtmp := new(Request)
		err := c.umrs.Unmarshal(c.rwc, rtmp)
		if err != nil {
			return err
		}
		m.Method = rtmp.GetServiceMethod()
		m.Id = rtmp.GetSeq()
	case codec.Response:
		rtmp := new(Response)
		err := c.umrs.Unmarshal(c.rwc, rtmp)
		if err != nil {
			return err
		}
		m.Method = rtmp.GetServiceMethod()
		m.Id = rtmp.GetSeq()
		m.Error = rtmp.GetError()
	case codec.Publication:
		io.Copy(c.buf, c.rwc)
	default:
		return fmt.Errorf("Unrecognised message type: %v", mt)
	}
	return nil
}

func (c *protoCodec) ReadBody(b interface{}) error {
	var reader io.Reader
	switch c.mt {
	case codec.Request, codec.Response:
		reader = c.rwc
	case codec.Publication:
		reader = c.buf
	default:
		return fmt.Errorf("Unrecognised message type: %v", c.mt)
	}
	if b != nil {
		return c.umrs.Unmarshal(reader, b.(proto.Message))
	}
	return nil
}

func NewCodec(rwc io.ReadWriteCloser) codec.Codec {
	return &protoCodec{
		buf:  bytes.NewBuffer(nil),
		rwc:  rwc,
		mrs:  &jsonpb.Marshaler{},
		umrs: &jsonpb.Unmarshaler{},
	}
}
