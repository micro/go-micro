package json

import (
	"bytes"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	jsoniter "github.com/json-iterator/go"
	"github.com/oxtoacart/bpool"
)

var (
	json = jsoniter.Config{
		EscapeHTML:             false,
		ValidateJsonRawMessage: false,
		SortMapKeys:            false,
	}.Froze()

	//json            = jsoniter.ConfigCompatibleWithStandardLibrary
	jsonpbMarshaler = &jsonpb.Marshaler{}

	// create buffer pool with 16 instances each preallocated with 256 bytes
	bufferPool = bpool.NewSizedBufferPool(16, 256)
)

type Marshaler struct{}

func (j Marshaler) Marshal(v interface{}) ([]byte, error) {
	if pb, ok := v.(proto.Message); ok {
		buf := bufferPool.Get()
		defer bufferPool.Put(buf)
		if err := jsonpbMarshaler.Marshal(buf, pb); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	}
	return json.Marshal(v)
}

func (j Marshaler) Unmarshal(d []byte, v interface{}) error {
	if pb, ok := v.(proto.Message); ok {
		return jsonpb.Unmarshal(bytes.NewReader(d), pb)
	}
	return json.Unmarshal(d, v)
}

func (j Marshaler) String() string {
	return "json"
}
