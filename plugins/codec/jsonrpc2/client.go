package jsonrpc2

import (
	"encoding/json"
	"errors"
	"io"
	"math"
	"reflect"
	"strconv"
	"sync"

	"github.com/micro/go-micro/v2/codec"
)

const seqNotify = math.MaxUint64

type clientCodec struct {
	dec *json.Decoder // for reading JSON values
	enc *json.Encoder // for writing JSON values
	c   io.Closer

	// temporary work space
	resp clientResponse

	// JSON-RPC responses include the request id but not the request method.
	// Package rpc expects both.
	// We save the request method in pending when sending a request
	// and then look it up by request ID when filling out the rpc Response.
	mutex   sync.Mutex             // protects pending
	pending map[interface{}]string // map request id to method name
}

func newClientCodec(conn io.ReadWriteCloser) *clientCodec {
	return &clientCodec{
		dec:     json.NewDecoder(conn),
		enc:     json.NewEncoder(conn),
		c:       conn,
		pending: make(map[interface{}]string),
	}
}

type clientRequest struct {
	Version string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
	ID      interface{} `json:"id,omitempty"`
}

func (c *clientCodec) Write(m *codec.Message, b interface{}) error {
	// If return error: it will be returned as is for this call.
	// Allow param to be only Array, Slice, Map or Struct.
	// When param is nil or uninitialized Map or Slice - omit "params".
	if b != nil {
		switch k := reflect.TypeOf(b).Kind(); k {
		case reflect.Map:
			if reflect.TypeOf(b).Key().Kind() == reflect.String {
				if reflect.ValueOf(b).IsNil() {
					b = nil
				}
			}
		case reflect.Slice:
			if reflect.ValueOf(b).IsNil() {
				b = nil
			}
		case reflect.Array, reflect.Struct:
		case reflect.Ptr:
			switch k := reflect.TypeOf(b).Elem().Kind(); k {
			case reflect.Map:
				if reflect.TypeOf(b).Elem().Key().Kind() == reflect.String {
					if reflect.ValueOf(b).Elem().IsNil() {
						b = nil
					}
				}
			case reflect.Slice:
				if reflect.ValueOf(b).Elem().IsNil() {
					b = nil
				}
			case reflect.Array, reflect.Struct:
			default:
				return NewError(errInternal.Code, "unsupported param type: Ptr to "+k.String())
			}
		default:
			return NewError(errInternal.Code, "unsupported param type: "+k.String())
		}
	}

	var req clientRequest

	i, _ := strconv.ParseInt(m.Id, 10, 64)

	if uint64(i) != seqNotify {
		c.mutex.Lock()
		c.pending[m.Id] = m.Endpoint
		c.mutex.Unlock()
		req.ID = m.Id
	}

	req.Version = "2.0"
	req.Method = m.Endpoint
	req.Params = b
	if err := c.enc.Encode(&req); err != nil {
		return NewError(errInternal.Code, err.Error())
	}
	return nil
}

type clientResponse struct {
	Version string           `json:"jsonrpc"`
	ID      interface{}      `json:"id"`
	Result  *json.RawMessage `json:"result,omitempty"`
	Error   *Error           `json:"error,omitempty"`
}

func (r *clientResponse) reset() {
	r.Version = ""
	r.ID = nil
	r.Result = nil
	r.Error = nil
}

func (r *clientResponse) UnmarshalJSON(raw []byte) error {
	r.reset()
	type resp *clientResponse
	if err := json.Unmarshal(raw, resp(r)); err != nil {
		return errors.New("bad response: " + string(raw))
	}

	var o = make(map[string]*json.RawMessage)
	if err := json.Unmarshal(raw, &o); err != nil {
		return errors.New("bad response: " + string(raw))
	}
	_, okVer := o["jsonrpc"]
	_, okID := o["id"]
	_, okRes := o["result"]
	_, okErr := o["error"]
	if !okVer || !okID || !(okRes || okErr) || (okRes && okErr) || len(o) > 3 {
		return errors.New("bad response: " + string(raw))
	}
	if r.Version != "2.0" {
		return errors.New("bad response: " + string(raw))
	}
	if okRes && r.Result == nil {
		r.Result = &null
	}
	if okErr {
		if o["error"] == nil {
			return errors.New("bad response: " + string(raw))
		}
		oe := make(map[string]*json.RawMessage)
		if err := json.Unmarshal(*o["error"], &oe); err != nil {
			return errors.New("bad response: " + string(raw))
		}
		if oe["code"] == nil || oe["message"] == nil {
			return errors.New("bad response: " + string(raw))
		}
		if _, ok := oe["data"]; (!ok && len(oe) > 2) || len(oe) > 3 {
			return errors.New("bad response: " + string(raw))
		}
	}
	if o["id"] == nil && !okErr {
		return errors.New("bad response: " + string(raw))
	}

	return nil
}

func (c *clientCodec) ReadHeader(m *codec.Message) error {
	// If return err:
	// - io.EOF will became ErrShutdown or io.ErrUnexpectedEOF
	// - it will be returned as is for all pending calls
	// - client will be shutdown
	// So, return io.EOF as is, return *Error for all other errors.
	c.resp.reset()
	if err := c.dec.Decode(&c.resp); err != nil {
		if err == io.EOF {
			return err
		}
		return NewError(errInternal.Code, err.Error())
	}
	if c.resp.ID == nil {
		return c.resp.Error
	}

	c.mutex.Lock()
	m.Endpoint = c.pending[c.resp.ID]
	delete(c.pending, c.resp.ID)
	c.mutex.Unlock()

	m.Error = ""
	m.Id = c.resp.ID.(string)
	if c.resp.Error != nil {
		m.Error = c.resp.Error.Error()
	}
	return nil
}

func (c *clientCodec) ReadBody(x interface{}) error {
	// If x!=nil and return error e:
	// - this call get e.Error() appended to "reading body "
	// - other pending calls get error as is XXX actually other calls
	//   shouldn't be affected by this error at all, so let's at least
	//   provide different error message for other calls
	if x == nil {
		return nil
	}
	if err := json.Unmarshal(*c.resp.Result, x); err != nil {
		e := NewError(errInternal.Code, err.Error())
		e.Data = NewError(errInternal.Code, "some other Call failed to unmarshal Reply")
		return e
	}
	return nil
}

func (c *clientCodec) Close() error {
	return c.c.Close()
}
