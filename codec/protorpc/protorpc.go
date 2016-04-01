package protorpc

import (
	"bytes"
	"fmt"
	"io"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/micro/go-micro/codec"
)

type flusher interface {
	Flush() error
}

type protoCodec struct {
	sync.Mutex
	rwc io.ReadWriteCloser
	mt  codec.MessageType
	buf *bytes.Buffer
}

func (c *protoCodec) Close() error {
	c.buf.Reset()
	return c.rwc.Close()
}

func (c *protoCodec) String() string {
	return "proto-rpc"
}

func (c *protoCodec) Write(m *codec.Message, b interface{}) error {
	switch m.Type {
	case codec.Request:
		c.Lock()
		defer c.Unlock()
		// This is protobuf, of course we copy it.
		pbr := &Request{ServiceMethod: &m.Method, Seq: &m.Id}
		data, err := proto.Marshal(pbr)
		if err != nil {
			return err
		}
		_, err = WriteNetString(c.rwc, data)
		if err != nil {
			return err
		}
		// Of course this is a protobuf! Trust me or detonate the program.
		data, err = proto.Marshal(b.(proto.Message))
		if err != nil {
			return err
		}
		_, err = WriteNetString(c.rwc, data)
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
		data, err := proto.Marshal(rtmp)
		if err != nil {
			return err
		}
		_, err = WriteNetString(c.rwc, data)
		if err != nil {
			return err
		}
		if pb, ok := b.(proto.Message); ok {
			data, err = proto.Marshal(pb)
			if err != nil {
				return err
			}
		} else {
			data = nil
		}
		_, err = WriteNetString(c.rwc, data)
		if err != nil {
			return err
		}
		if flusher, ok := c.rwc.(flusher); ok {
			err = flusher.Flush()
		}
	case codec.Publication:
		data, err := proto.Marshal(b.(proto.Message))
		if err != nil {
			return err
		}
		c.rwc.Write(data)
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
		data, err := ReadNetString(c.rwc)
		if err != nil {
			return err
		}
		rtmp := new(Request)
		err = proto.Unmarshal(data, rtmp)
		if err != nil {
			return err
		}
		m.Method = rtmp.GetServiceMethod()
		m.Id = rtmp.GetSeq()
	case codec.Response:
		data, err := ReadNetString(c.rwc)
		if err != nil {
			return err
		}
		rtmp := new(Response)
		err = proto.Unmarshal(data, rtmp)
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
	var data []byte
	switch c.mt {
	case codec.Request, codec.Response:
		var err error
		data, err = ReadNetString(c.rwc)
		if err != nil {
			return err
		}
	case codec.Publication:
		data = c.buf.Bytes()
	default:
		return fmt.Errorf("Unrecognised message type: %v", c.mt)
	}
	if b != nil {
		return proto.Unmarshal(data, b.(proto.Message))
	}
	return nil
}

func NewCodec(rwc io.ReadWriteCloser) codec.Codec {
	return &protoCodec{
		buf: bytes.NewBuffer(nil),
		rwc: rwc,
	}
}
