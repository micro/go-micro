package msgp

import (
	"errors"

	"github.com/micro/go-micro/broker"
	"github.com/micro/go-micro/broker/codec"
)

type msgpCodec struct{}

func (n msgpCodec) Marshal(v interface{}) ([]byte, error) {
	msg, ok := v.(*broker.Message)
	if !ok {
		return nil, errors.New("invalid message")
	}

	out := make([]byte, msg.Msgsize())
	return msg.MarshalMsg(out)
}

func (n msgpCodec) Unmarshal(d []byte, v interface{}) error {
	msg, ok := v.(*broker.Message)
	if !ok {
		return errors.New("invalid message")
	}
	_, err := msg.UnmarshalMsg(d)
	return err
}

func (n msgpCodec) String() string {
	return "msgp"
}

func NewCodec() codec.Codec {
	return msgpCodec{}
}
