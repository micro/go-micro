package jsonrpc2

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/asim/go-micro/v3/codec"
)

type serverCodec struct {
	encmutex sync.Mutex    // protects enc
	dec      *json.Decoder // for reading JSON values
	enc      *json.Encoder // for writing JSON values
	c        io.Closer

	// temporary work space
	req serverRequest

	// JSON-RPC clients can use arbitrary json values as request IDs.
	// Package rpc expects uint64 request IDs.
	// We assign uint64 sequence numbers to incoming requests
	// but save the original request ID in the pending map.
	// When rpc responds, we use the sequence number in
	// the response to find the original request ID.
	mutex   sync.Mutex // protects seq, pending
	seq     uint64
	pending map[interface{}]*json.RawMessage
}

func newServerCodec(conn io.ReadWriteCloser) *serverCodec {
	return &serverCodec{
		dec:     json.NewDecoder(conn),
		enc:     json.NewEncoder(conn),
		c:       conn,
		pending: make(map[interface{}]*json.RawMessage),
	}
}

type serverRequest struct {
	Version string           `json:"jsonrpc"`
	Method  string           `json:"method"`
	Params  *json.RawMessage `json:"params"`
	ID      *json.RawMessage `json:"id"`
}

func (r *serverRequest) reset() {
	r.Version = ""
	r.Method = ""
	r.Params = nil
	r.ID = nil
}

func (r *serverRequest) UnmarshalJSON(raw []byte) error {
	r.reset()
	type req *serverRequest
	if err := json.Unmarshal(raw, req(r)); err != nil {
		return errors.New("bad request")
	}

	var o = make(map[string]*json.RawMessage)
	if err := json.Unmarshal(raw, &o); err != nil {
		return errors.New("bad request")
	}
	if o["jsonrpc"] == nil || o["method"] == nil {
		return errors.New("bad request")
	}
	_, okID := o["id"]
	_, okParams := o["params"]
	if len(o) == 3 && !(okID || okParams) || len(o) == 4 && !(okID && okParams) || len(o) > 4 {
		return errors.New("bad request")
	}
	if r.Version != "2.0" {
		return errors.New("bad request")
	}
	if okParams {
		if r.Params == nil || len(*r.Params) == 0 {
			return errors.New("bad request")
		}
		switch []byte(*r.Params)[0] {
		case '[', '{':
		default:
			return errors.New("bad request")
		}
	}
	if okID && r.ID == nil {
		r.ID = &null
	}
	if okID {
		if len(*r.ID) == 0 {
			return errors.New("bad request")
		}
		switch []byte(*r.ID)[0] {
		case 't', 'f', '{', '[':
			return errors.New("bad request")
		}
	}

	return nil
}

type serverResponse struct {
	Version string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id"`
	Result  interface{}      `json:"result,omitempty"`
	Error   interface{}      `json:"error,omitempty"`
}

func (c *serverCodec) ReadHeader(m *codec.Message) (err error) {
	// If return error:
	// - codec will be closed
	// So, try to send error reply to client before returning error.
	c.req.reset()
	var raw json.RawMessage
	if err := c.dec.Decode(&raw); err != nil {
		c.encmutex.Lock()
		c.enc.Encode(serverResponse{Version: "2.0", ID: &null, Error: errParse})
		c.encmutex.Unlock()
		return err
	}

	if err := json.Unmarshal(raw, &c.req); err != nil {
		if err.Error() == "bad request" {
			c.encmutex.Lock()
			c.enc.Encode(serverResponse{Version: "2.0", ID: &null, Error: errRequest})
			c.encmutex.Unlock()
		}
		return err
	}

	m.Endpoint = c.req.Method

	// JSON request id can be any JSON value;
	// RPC package expects uint64.  Translate to
	// internal uint64 and save JSON on the side.
	c.mutex.Lock()
	c.seq++
	c.pending[c.seq] = c.req.ID
	c.req.ID = nil
	m.Id = fmt.Sprintf("%d", c.seq)
	c.mutex.Unlock()

	return nil
}

func (c *serverCodec) ReadBody(x interface{}) error {
	// If x!=nil and return error e:
	// - WriteResponse() will be called with e.Error() in r.Error
	if x == nil {
		return nil
	}
	if c.req.Params == nil {
		return nil
	}
	if err := json.Unmarshal(*c.req.Params, x); err != nil {
		return NewError(errParams.Code, err.Error())
	}
	return nil
}

var null = json.RawMessage([]byte("null"))

func (c *serverCodec) Write(m *codec.Message, x interface{}) error {
	// If return error: nothing happens.
	// In r.Error will be "" or .Error() of error returned by:
	// - ReadRequestBody()
	// - called RPC method
	c.mutex.Lock()
	b, ok := c.pending[m.Id]
	if !ok {
		c.mutex.Unlock()
		fmt.Println("invalid sequence number in response", m.Id)
		return errors.New("invalid sequence number in response")
	}
	c.mutex.Unlock()

	if replies, ok := x.(*[]*json.RawMessage); m.Endpoint == "JSONRPC2.Batch" && ok {
		if len(*replies) == 0 {
			return nil
		}
		c.encmutex.Lock()
		defer c.encmutex.Unlock()
		return c.enc.Encode(replies)
	}

	if b == nil {
		// Notification. Do not respond.
		return nil
	}
	resp := serverResponse{Version: "2.0", ID: b}
	if m.Error == "" {
		if x == nil {
			resp.Result = &null
		} else {
			resp.Result = x
		}
	} else if m.Error[0] == '{' && m.Error[len(m.Error)-1] == '}' {
		// Well… this check for '{'…'}' isn't too strict, but I
		// suppose we're trusting our own RPC methods (this way they
		// can force sending wrong reply or many replies instead
		// of one) and normal errors won't be formatted this way.
		raw := json.RawMessage(m.Error)
		resp.Error = &raw
	} else {
		raw := json.RawMessage(newError(m.Error).Error())
		resp.Error = &raw
	}
	c.encmutex.Lock()
	defer c.encmutex.Unlock()
	return c.enc.Encode(resp)
}

func (c *serverCodec) Close() error {
	return c.c.Close()
}
