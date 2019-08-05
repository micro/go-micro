package jsonrpc

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/micro/go-micro/codec"
)

type serverCodec struct {
	dec *json.Decoder // for reading JSON values
	enc *json.Encoder // for writing JSON values
	c   io.Closer

	// temporary work space
	req  serverRequest
	resp serverResponse
}

type serverRequest struct {
	Method string           `json:"method"`
	Params *json.RawMessage `json:"params"`
	ID     interface{}      `json:"id"`
}

type serverResponse struct {
	ID     interface{} `json:"id"`
	Result interface{} `json:"result"`
	Error  interface{} `json:"error"`
}

func newServerCodec(conn io.ReadWriteCloser) *serverCodec {
	return &serverCodec{
		dec: json.NewDecoder(conn),
		enc: json.NewEncoder(conn),
		c:   conn,
	}
}

func (r *serverRequest) reset() {
	r.Method = ""
	if r.Params != nil {
		*r.Params = (*r.Params)[0:0]
	}
	if r.ID != nil {
		r.ID = nil
	}
}

func (c *serverCodec) ReadHeader(m *codec.Message) error {
	c.req.reset()
	if err := c.dec.Decode(&c.req); err != nil {
		return err
	}
	m.Method = c.req.Method
	m.Id = fmt.Sprintf("%v", c.req.ID)
	c.req.ID = nil
	return nil
}

func (c *serverCodec) ReadBody(x interface{}) error {
	if x == nil {
		return nil
	}
	var params [1]interface{}
	params[0] = x
	return json.Unmarshal(*c.req.Params, &params)
}

var null = json.RawMessage([]byte("null"))

func (c *serverCodec) Write(m *codec.Message, x interface{}) error {
	var resp serverResponse
	resp.ID = m.Id
	resp.Result = x
	if m.Error == "" {
		resp.Error = nil
	} else {
		resp.Error = m.Error
	}
	return c.enc.Encode(resp)
}

func (c *serverCodec) Close() error {
	return c.c.Close()
}
