// Package grpc provides a grpc codec
package grpc

import (
	"encoding/json"
	"errors"
	"io"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/micro/go-micro/codec"
)

type Codec struct {
	Conn        io.ReadWriteCloser
	ContentType string
}

func (c *Codec) ReadHeader(m *codec.Message, t codec.MessageType) error {
	if ct := m.Header["Content-Type"]; len(ct) > 0 {
		c.ContentType = ct
	}

	if ct := m.Header["content-type"]; len(ct) > 0 {
		c.ContentType = ct
	}

	// service method
	path := m.Header[":path"]
	if len(path) == 0 || path[0] != '/' {
		m.Target = m.Header["X-Micro-Service"]
		m.Endpoint = m.Header["X-Micro-Endpoint"]
	} else {
		// [ , a.package.Foo, Bar]
		parts := strings.Split(path, "/")
		if len(parts) != 3 {
			return errors.New("Unknown request path")
		}
		service := strings.Split(parts[1], ".")
		m.Endpoint = strings.Join([]string{service[len(service)-1], parts[2]}, ".")
		m.Target = strings.Join(service[:len(service)-1], ".")
	}

	return nil
}

func (c *Codec) ReadBody(b interface{}) error {
	// no body
	if b == nil {
		return nil
	}

	_, buf, err := decode(c.Conn)
	if err != nil {
		return err
	}

	switch c.ContentType {
	case "application/grpc+json":
		return json.Unmarshal(buf, b)
	case "application/grpc+proto", "application/grpc":
		return proto.Unmarshal(buf, b.(proto.Message))
	}

	return errors.New("Unsupported Content-Type")
}

func (c *Codec) Write(m *codec.Message, b interface{}) error {
	var buf []byte
	var err error

	if ct := m.Header["Content-Type"]; len(ct) > 0 {
		c.ContentType = ct
	}

	if ct := m.Header["content-type"]; len(ct) > 0 {
		c.ContentType = ct
	}

	m.Header["Trailer"] = "grpc-status, grpc-message"

	switch c.ContentType {
	case "application/grpc+json":
		buf, err = json.Marshal(b)
	case "application/grpc+proto", "application/grpc":
		pb, ok := b.(proto.Message)
		if ok {
			buf, err = proto.Marshal(pb)
		}
	default:
		err = errors.New("Unsupported Content-Type")
	}

	if err != nil {
		m.Header["grpc-status"] = "8"
		m.Header["grpc-message"] = err.Error()
		return err
	}

	m.Header["grpc-status"] = "0"
	m.Header["grpc-message"] = ""

	return encode(0, buf, c.Conn)
}

func (c *Codec) Close() error {
	return c.Conn.Close()
}

func (c *Codec) String() string {
	return "grpc"
}

func NewCodec(c io.ReadWriteCloser) codec.Codec {
	return &Codec{
		Conn:        c,
		ContentType: "application/grpc",
	}
}
