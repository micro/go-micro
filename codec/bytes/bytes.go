// Package bytes provides a bytes codec which does not encode or decode anything
package bytes

import (
	"fmt"
	"io"
	"io/ioutil"

	"github.com/micro/go-micro/codec"
)

type Codec struct {
	Conn io.ReadWriteCloser
}

func (c *Codec) ReadHeader(m *codec.Message, t codec.MessageType) error {
	return nil
}

func (c *Codec) ReadBody(b interface{}) error {
	v, ok := b.(*[]byte)
	if !ok {
		return fmt.Errorf("failed to read body: %v is not type of *[]byte", b)
	}

	// read bytes
	buf, err := ioutil.ReadAll(c.Conn)
	if err != nil {
		return err
	}

	// set bytes
	*v = buf
	return nil
}

func (c *Codec) Write(m *codec.Message, b interface{}) error {
	v, ok := b.(*[]byte)
	if !ok {
		return fmt.Errorf("failed to write: %v is not type of *[]byte", b)
	}
	_, err := c.Conn.Write(*v)
	return err
}

func (c *Codec) Close() error {
	return c.Conn.Close()
}

func (c *Codec) String() string {
	return "bytes"
}

func NewCodec(c io.ReadWriteCloser) codec.Codec {
	return &Codec{
		Conn: c,
	}
}
