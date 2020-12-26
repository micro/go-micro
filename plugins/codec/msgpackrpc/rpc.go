// Package msgpackrpc provides a msgpack-rpc codec
package msgpackrpc

import (
	"errors"

	"github.com/tinylib/msgp/msgp"
)

// The msgpack-rpc specification: https://github.com/msgpack-rpc/msgpack-rpc/blob/master/spec.md
const (
	RequestType      = 0
	ResponseType     = 1
	NotificationType = 2

	RequestPackSize      = 4
	ResponsePackSize     = 4
	NotificationPackSize = 3
)

var (
	ErrBadPackSize      = errors.New("Bad pack size")
	ErrBadMessageType   = errors.New("Bad message type")
	ErrBadErrorType     = errors.New("Bad error type")
	ErrUnexpectedParams = errors.New("Unexpected params")
	ErrNotEncodable     = errors.New("Not encodable")
	ErrNotDecodable     = errors.New("Not decodable")
)

// decodeBody decodes the body of the message.
func decodeBody(r *msgp.Reader, v interface{}) error {
	b, ok := v.(msgp.Decodable)
	if !ok {
		return ErrNotDecodable
	}

	return msgp.Decode(r, b)
}

// Request is what the client can construct to be sent to the server.
// The params represents the body of the request.
type Request struct {
	ID     string
	Method string
	Body   interface{}

	hasBody bool
}

// EncodeMsg encodes the request to writer. The body is expected to
// be an encodable type.
func (r *Request) EncodeMsg(w *msgp.Writer) error {
	var bm msgp.Encodable

	if r.Body != nil {
		var ok bool
		bm, ok = r.Body.(msgp.Encodable)
		if !ok {
			return ErrNotEncodable
		}
	}

	var err error

	if err = w.WriteArrayHeader(RequestPackSize); err != nil {
		return err
	}

	if err = w.WriteInt(RequestType); err != nil {
		return err
	}

	if err = w.WriteString(r.ID); err != nil {
		return err
	}

	if err = w.WriteString(r.Method); err != nil {
		return err
	}

	// No body to encode. Write a zero-length params array.
	if bm == nil {
		return w.WriteArrayHeader(0)
	}

	// 1-item array containing the body.
	if err = w.WriteArrayHeader(1); err != nil {
		return err
	}

	return msgp.Encode(w, bm)
}

func (r *Request) DecodeMsg(mr *msgp.Reader) error {
	var bm msgp.Decodable

	if r.Body != nil {
		var ok bool
		bm, ok = r.Body.(msgp.Decodable)
		if !ok {
			return ErrNotDecodable
		}
	}

	if size, err := mr.ReadArrayHeader(); err != nil {
		return err
	} else if size != RequestPackSize {
		return ErrBadPackSize
	}

	if typ, err := mr.ReadInt(); err != nil {
		return err
	} else if typ != RequestType {
		return ErrBadMessageType
	}

	id, err := mr.ReadString()
	if err != nil {
		return err
	}

	r.ID = id

	method, err := mr.ReadString()
	if err != nil {
		return err
	}

	r.Method = method

	// The request body is packed in an array.
	l, err := mr.ReadArrayHeader()
	if err != nil {
		return err
	}

	if l > 1 {
		return ErrUnexpectedParams
	} else if l == 0 {
		return nil
	}

	r.hasBody = true

	// Skip decoding the body if no value is present to decode into.
	// The caller is expected to decode the body or skip it.
	if bm != nil {
		return decodeBody(mr, bm)
	}

	return nil
}

type Response struct {
	ID    string
	Error string
	Body  interface{}

	hasBody bool
}

func (r *Response) EncodeMsg(w *msgp.Writer) error {
	var bm msgp.Encodable

	if r.Body != nil {
		var ok bool
		bm, ok = r.Body.(msgp.Encodable)
		if !ok {
			return ErrNotEncodable
		}
	}

	var err error

	if err = w.WriteArrayHeader(ResponsePackSize); err != nil {
		return err
	}

	if err = w.WriteInt(ResponseType); err != nil {
		return err
	}

	if err = w.WriteString(r.ID); err != nil {
		return err
	}

	// No error.
	if r.Error == "" {
		if err = w.WriteNil(); err != nil {
			return err
		}

		if bm != nil {
			return msgp.Encode(w, bm)
		}
	} else {
		if err = w.WriteString(r.Error); err != nil {
			return err
		}
	}

	// Write nil body.
	return w.WriteNil()
}

func (r *Response) DecodeMsg(mr *msgp.Reader) error {
	var bm msgp.Decodable

	if r.Body != nil {
		var ok bool
		bm, ok = r.Body.(msgp.Decodable)
		if !ok {
			return ErrNotDecodable
		}
	}

	if size, err := mr.ReadArrayHeader(); err != nil {
		return err
	} else if size != ResponsePackSize {
		return ErrBadPackSize
	}

	if typ, err := mr.ReadInt(); err != nil {
		return err
	} else if typ != ResponseType {
		return ErrBadMessageType
	}

	id, err := mr.ReadString()
	if err != nil {
		return err
	}

	r.ID = id

	// Error can be nil or a string.
	typ, err := mr.NextType()
	if err != nil {
		return err
	}

	switch typ {
	case msgp.StrType:
		s, err := mr.ReadString()
		if err != nil {
			return err
		}
		r.Error = s

	case msgp.NilType:
		if err := mr.ReadNil(); err != nil {
			return err
		}
		r.Error = ""

	default:
		return ErrBadErrorType
	}

	// Body can be nil.
	typ, err = mr.NextType()
	if err != nil {
		return err
	}

	if typ == msgp.NilType {
		r.hasBody = false
		return mr.ReadNil()
	}

	r.hasBody = true

	// Skip decoding the body if no value is present to decode into.
	// The caller is expected to read the body or skip it.
	if bm != nil {
		return decodeBody(mr, bm)
	}

	return nil

}

type Notification struct {
	Method string
	Body   interface{}

	hasBody bool
}

// EncodeMsg encodes the notification to writer. The body is expected to
// be an encodable type.
func (n *Notification) EncodeMsg(w *msgp.Writer) error {
	var bm msgp.Encodable

	if n.Body != nil {
		var ok bool
		bm, ok = n.Body.(msgp.Encodable)
		if !ok {
			return ErrNotEncodable
		}
	}

	var err error

	if err = w.WriteArrayHeader(NotificationPackSize); err != nil {
		return err
	}

	if err = w.WriteInt(NotificationType); err != nil {
		return err
	}

	if err = w.WriteString(n.Method); err != nil {
		return err
	}

	// No body to encode. Write a zero-length params array.
	if bm == nil {
		return w.WriteArrayHeader(0)
	}

	// 1-item array containing the body.
	if err = w.WriteArrayHeader(1); err != nil {
		return err
	}

	return msgp.Encode(w, bm)
}

func (n *Notification) DecodeMsg(mr *msgp.Reader) error {
	var bm msgp.Decodable

	if n.Body != nil {
		var ok bool
		bm, ok = n.Body.(msgp.Decodable)
		if !ok {
			return ErrNotDecodable
		}
	}

	if size, err := mr.ReadArrayHeader(); err != nil {
		return err
	} else if size != NotificationPackSize {
		return ErrBadPackSize
	}

	if typ, err := mr.ReadInt(); err != nil {
		return err
	} else if typ != NotificationType {
		return ErrBadMessageType
	}

	method, err := mr.ReadString()
	if err != nil {
		return err
	}

	n.Method = method

	// The notification body is packed in an array.
	l, err := mr.ReadArrayHeader()
	if err != nil {
		return err
	}

	if l > 1 {
		return ErrUnexpectedParams
	} else if l == 0 {
		return nil
	}

	n.hasBody = true

	// Skip decoding the body if no value is present to decode into.
	// The caller is expected to decode the body or skip it.
	if bm != nil {
		return decodeBody(mr, bm)
	}

	return nil
}
