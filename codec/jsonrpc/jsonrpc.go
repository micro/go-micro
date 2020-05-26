// Package jsonrpc provides a json-rpc 1.0 codec
package jsonrpc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/micro/go-micro/v2/codec"
)

type jsonCodec struct {
	buf *bytes.Buffer
	mt  codec.MessageType
	rwc io.ReadWriteCloser
	c   *clientCodec
	s   *serverCodec
}

func (j *jsonCodec) Close() error {
	j.buf.Reset()
	return j.rwc.Close()
}

func (j *jsonCodec) String() string {
	return "json-rpc"
}

func (j *jsonCodec) Write(m *codec.Message, b interface{}) error {
	switch m.Type {
	case codec.Request:
		return j.c.Write(m, b)
	case codec.Response, codec.Error:
		return j.s.Write(m, b)
	case codec.Event:
		data, err := json.Marshal(b)
		if err != nil {
			return err
		}
		_, err = j.rwc.Write(data)
		return err
	default:
		return fmt.Errorf("Unrecognised message type: %v", m.Type)
	}
}

func (j *jsonCodec) ReadHeader(m *codec.Message, mt codec.MessageType) error {
	j.buf.Reset()
	j.mt = mt

	switch mt {
	case codec.Request:
		return j.s.ReadHeader(m)
	case codec.Response:
		return j.c.ReadHeader(m)
	case codec.Event:
		_, err := io.Copy(j.buf, j.rwc)
		return err
	default:
		return fmt.Errorf("Unrecognised message type: %v", mt)
	}
}

func (j *jsonCodec) ReadBody(b interface{}) error {
	switch j.mt {
	case codec.Request:
		return j.s.ReadBody(b)
	case codec.Response:
		return j.c.ReadBody(b)
	case codec.Event:
		if b != nil {
			return json.Unmarshal(j.buf.Bytes(), b)
		}
	default:
		return fmt.Errorf("Unrecognised message type: %v", j.mt)
	}
	return nil
}

func NewCodec(rwc io.ReadWriteCloser) codec.Codec {
	return &jsonCodec{
		buf: bytes.NewBuffer(nil),
		rwc: rwc,
		c:   newClientCodec(rwc),
		s:   newServerCodec(rwc),
	}
}
