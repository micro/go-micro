package bsonrpc

import (
	"io"

	"github.com/asim/go-bson"
	"github.com/micro/go-micro/v2/codec"
)

type clientCodec struct {
	rwc io.ReadWriteCloser
}

type serverCodec struct {
	rwc io.ReadWriteCloser
}

type request struct {
	ServiceMethod string
	Seq           string
}

type response struct {
	ServiceMethod string
	Seq           string
	Error         string
}

func (c *clientCodec) Write(m *codec.Message, body interface{}) error {
	if err := bson.MarshalToStream(c.rwc, &request{
		ServiceMethod: m.Endpoint,
		Seq:           m.Id,
	}); err != nil {
		return err
	}
	if err := bson.MarshalToStream(c.rwc, body); err != nil {
		return err
	}
	return nil
}

func (c *clientCodec) ReadHeader(m *codec.Message) error {
	r := &response{}
	if err := bson.UnmarshalFromStream(c.rwc, r); err != nil {
		return err
	}
	m.Id = r.Seq
	m.Endpoint = r.ServiceMethod
	m.Error = r.Error
	return nil
}

func (c *clientCodec) ReadBody(body interface{}) error {
	if body == nil {
		return nil
	}
	return bson.UnmarshalFromStream(c.rwc, body)
}

func (c *clientCodec) Close() error {
	return c.rwc.Close()
}

func (s *serverCodec) ReadHeader(m *codec.Message) error {
	r := &request{}
	if err := bson.UnmarshalFromStream(s.rwc, r); err != nil {
		return err
	}
	m.Id = r.Seq
	m.Endpoint = r.ServiceMethod
	return nil
}

func (s *serverCodec) ReadBody(body interface{}) error {
	if body == nil {
		return nil
	}
	return bson.UnmarshalFromStream(s.rwc, body)
}

func (s *serverCodec) Write(m *codec.Message, body interface{}) error {
	if err := bson.MarshalToStream(s.rwc, &response{
		ServiceMethod: m.Endpoint,
		Seq:           m.Id,
		Error:         m.Error,
	}); err != nil {
		return err
	}
	if err := bson.MarshalToStream(s.rwc, body); err != nil {
		return err
	}
	return nil
}

func (s *serverCodec) Close() error {
	return s.rwc.Close()
}

func newClientCodec(rwc io.ReadWriteCloser) *clientCodec {
	return &clientCodec{rwc}
}

func newServerCodec(rwc io.ReadWriteCloser) *serverCodec {
	return &serverCodec{rwc}
}
