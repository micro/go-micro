package jsonrpc

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/micro/go-micro/v2/codec"
)

type clientCodec struct {
	dec *json.Decoder // for reading JSON values
	enc *json.Encoder // for writing JSON values
	c   io.Closer

	// temporary work space
	req  clientRequest
	resp clientResponse

	sync.Mutex
	pending map[interface{}]string
}

type clientRequest struct {
	Method string         `json:"method"`
	Params [1]interface{} `json:"params"`
	ID     interface{}    `json:"id"`
}

type clientResponse struct {
	ID     interface{}      `json:"id"`
	Result *json.RawMessage `json:"result"`
	Error  interface{}      `json:"error"`
}

func newClientCodec(conn io.ReadWriteCloser) *clientCodec {
	return &clientCodec{
		dec:     json.NewDecoder(conn),
		enc:     json.NewEncoder(conn),
		c:       conn,
		pending: make(map[interface{}]string),
	}
}

func (c *clientCodec) Write(m *codec.Message, b interface{}) error {
	c.Lock()
	c.pending[m.Id] = m.Method
	c.Unlock()
	c.req.Method = m.Method
	c.req.Params[0] = b
	c.req.ID = m.Id
	return c.enc.Encode(&c.req)
}

func (r *clientResponse) reset() {
	r.ID = 0
	r.Result = nil
	r.Error = nil
}

func (c *clientCodec) ReadHeader(m *codec.Message) error {
	c.resp.reset()
	if err := c.dec.Decode(&c.resp); err != nil {
		return err
	}

	c.Lock()
	m.Method = c.pending[c.resp.ID]
	delete(c.pending, c.resp.ID)
	c.Unlock()

	m.Error = ""
	m.Id = fmt.Sprintf("%v", c.resp.ID)
	if c.resp.Error != nil {
		x, ok := c.resp.Error.(string)
		if !ok {
			return fmt.Errorf("invalid error %v", c.resp.Error)
		}
		if x == "" {
			x = "unspecified error"
		}
		m.Error = x
	}
	return nil
}

func (c *clientCodec) ReadBody(x interface{}) error {
	if x == nil || c.resp.Result == nil {
		return nil
	}
	return json.Unmarshal(*c.resp.Result, x)
}

func (c *clientCodec) Close() error {
	return c.c.Close()
}
