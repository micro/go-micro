package http

import (
	"encoding/json"

	"github.com/golang/protobuf/proto"
	"github.com/asim/go-micro/v3/codec"
	"github.com/asim/go-micro/v3/codec/jsonrpc"
	"github.com/asim/go-micro/v3/codec/protorpc"
)

type jsonCodec struct{}

type protoCodec struct{}

type Codec interface {
	Marshal(v interface{}) ([]byte, error)
	Unmarshal(b []byte, v interface{}) error
	String() string
}

var (
	defaultHTTPCodecs = map[string]Codec{
		"application/json":         jsonCodec{},
		"application/proto":        protoCodec{},
		"application/protobuf":     protoCodec{},
		"application/octet-stream": protoCodec{},
	}

	defaultRPCCodecs = map[string]codec.NewCodec{
		"application/json":         jsonrpc.NewCodec,
		"application/json-rpc":     jsonrpc.NewCodec,
		"application/protobuf":     protorpc.NewCodec,
		"application/proto-rpc":    protorpc.NewCodec,
		"application/octet-stream": protorpc.NewCodec,
	}
)

func (protoCodec) Marshal(v interface{}) ([]byte, error) {
	return proto.Marshal(v.(proto.Message))
}

func (protoCodec) Unmarshal(data []byte, v interface{}) error {
	return proto.Unmarshal(data, v.(proto.Message))
}

func (protoCodec) String() string {
	return "proto"
}

func (jsonCodec) Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func (jsonCodec) Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

func (jsonCodec) String() string {
	return "json"
}
