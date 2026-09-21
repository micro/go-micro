package json

import (
	"encoding/json"

	"google.golang.org/protobuf/proto"
)

type Marshaler struct{ Options Options }

func (j Marshaler) Marshal(v interface{}) ([]byte, error) {
	if pb, ok := v.(proto.Message); ok {
		data, err := j.Options.MarshalOptions.Marshal(pb)
		if err != nil || !j.Options.Int64AsNumber {
			return data, err
		}
		return j.Options.numericIntegers(data, pb.ProtoReflect().Descriptor())
	}
	return json.Marshal(v)
}

func (j Marshaler) Unmarshal(d []byte, v interface{}) error {
	if pb, ok := v.(proto.Message); ok {
		return j.Options.UnmarshalOptions.Unmarshal(d, pb)
	}
	return json.Unmarshal(d, v)
}

func (j Marshaler) String() string {
	return "json"
}
