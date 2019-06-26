package proto

import (
	"github.com/golang/protobuf/proto"
)

type Marshaler struct{}

func (Marshaler) Marshal(v interface{}) ([]byte, error) {
	return proto.Marshal(v.(proto.Message))
}

func (Marshaler) Unmarshal(data []byte, v interface{}) error {
	return proto.Unmarshal(data, v.(proto.Message))
}

func (Marshaler) String() string {
	return "proto"
}
