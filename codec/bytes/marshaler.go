package bytes

import (
	"errors"
)

type Marshaler struct{}

type Message struct {
	Header map[string]string
	Body   []byte
}

func (n Marshaler) Marshal(v interface{}) ([]byte, error) {
	switch v.(type) {
	case *[]byte:
		ve := v.(*[]byte)
		return *ve, nil
	case []byte:
		return v.([]byte), nil
	case *Message:
		return v.(*Message).Body, nil
	}
	return nil, errors.New("invalid message")
}

func (n Marshaler) Unmarshal(d []byte, v interface{}) error {
	switch v.(type) {
	case *[]byte:
		ve := v.(*[]byte)
		*ve = d
	case *Message:
		ve := v.(*Message)
		ve.Body = d
	}
	return errors.New("invalid message")
}

func (n Marshaler) String() string {
	return "bytes"
}
