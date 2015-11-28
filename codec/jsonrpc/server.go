package jsonrpc

import (
	"encoding/json"
	"errors"
	"io"
	"sync"

	"github.com/micro/go-micro/codec"
)

type serverCodec struct {
	dec *json.Decoder // for reading JSON values
	enc *json.Encoder // for writing JSON values
	c   io.Closer

	// temporary work space
	req  serverRequest
	resp serverResponse

	sync.Mutex
	seq     uint64
	pending map[uint64]*json.RawMessage
}

type serverRequest struct {
	Method string           `json:"method"`
	Params *json.RawMessage `json:"params"`
	ID     *json.RawMessage `json:"id"`
}

type serverResponse struct {
	ID     *json.RawMessage `json:"id"`
	Result interface{}      `json:"result"`
	Error  interface{}      `json:"error"`
}

func newServerCodec(conn io.ReadWriteCloser) *serverCodec {
	return &serverCodec{
		dec:     json.NewDecoder(conn),
		enc:     json.NewEncoder(conn),
		c:       conn,
		pending: make(map[uint64]*json.RawMessage),
	}
}

func (r *serverRequest) reset() {
	r.Method = ""
	if r.Params != nil {
		*r.Params = (*r.Params)[0:0]
	}
	if r.ID != nil {
		*r.ID = (*r.ID)[0:0]
	}
}

func (c *serverCodec) ReadHeader(m *codec.Message) error {
	c.req.reset()
	if err := c.dec.Decode(&c.req); err != nil {
		return err
	}
	m.Method = c.req.Method

	c.Lock()
	c.seq++
	c.pending[c.seq] = c.req.ID
	c.req.ID = nil
	m.Id = c.seq
	c.Unlock()

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
	c.Lock()
	b, ok := c.pending[m.Id]
	if !ok {
		c.Unlock()
		return errors.New("invalid sequence number in response")
	}
	c.Unlock()

	if b == nil {
		// Invalid request so no id.  Use JSON null.
		b = &null
	}
	resp.ID = b
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
