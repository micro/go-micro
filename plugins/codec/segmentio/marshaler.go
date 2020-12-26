package json

import (
	"bytes"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/oxtoacart/bpool"
	"github.com/segmentio/encoding/json"
)

var jsonpbMarshaler = &jsonpb.Marshaler{}
var jsonpbUnmarshaler = &jsonpb.Unmarshaler{}

// create buffer pool with 16 instances each preallocated with 256 bytes
var bufferPool = bpool.NewSizedBufferPool(16, 256)
var bytesPool = bpool.NewBytePool(16*256, 256)

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
	buf := bytesPool.Get()
	defer bytesPool.Put(buf)
	return json.Append(buf[:0], v, 0)
}

func (j Marshaler) Unmarshal(d []byte, v interface{}) error {
	if pb, ok := v.(proto.Message); ok {
		return jsonpbUnmarshaler.Unmarshal(bytes.NewReader(d), pb)
	}
	if _, err := json.Parse(d, v, json.ZeroCopy); err != nil {
		return err
	}
	return nil
}

func (j Marshaler) String() string {
	return "json"
}
