package noop

import (
	"errors"

	"github.com/micro/go-micro/broker"
	"github.com/micro/go-micro/broker/codec"
)

type noopCodec struct{}

func (n noopCodec) Marshal(v interface{}) ([]byte, error) {
	msg, ok := v.(*broker.Message)
	if !ok {
		return nil, errors.New("invalid message")
	}
	return msg.Body, nil
}

func (n noopCodec) Unmarshal(d []byte, v interface{}) error {
	msg, ok := v.(*broker.Message)
	if !ok {
		return errors.New("invalid message")
	}
	msg.Body = d
	return nil
}

func (n noopCodec) String() string {
	return "noop"
}

func NewCodec() codec.Codec {
	return noopCodec{}
}
