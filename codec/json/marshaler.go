package json

import (
	"encoding/json"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

var protojsonMarshaler = protojson.MarshalOptions{
	EmitUnpopulated: false,
}

type Marshaler struct{}

func (j Marshaler) Marshal(v interface{}) ([]byte, error) {
	if pb, ok := v.(proto.Message); ok {
		return protojsonMarshaler.Marshal(pb)
	}
	return json.Marshal(v)
}

func (j Marshaler) Unmarshal(d []byte, v interface{}) error {
	if pb, ok := v.(proto.Message); ok {
		return protojson.Unmarshal(d, pb)
	}
	return json.Unmarshal(d, v)
}

func (j Marshaler) String() string {
	return "json"
}
