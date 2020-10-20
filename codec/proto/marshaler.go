package proto

import (
	"bytes"

	"github.com/asim/go-micro/v3/codec"
	"github.com/golang/protobuf/proto"
	"github.com/oxtoacart/bpool"
)

// create buffer pool with 16 instances each preallocated with 256 bytes
var bufferPool = bpool.NewSizedBufferPool(16, 256)

type Marshaler struct{}

func (Marshaler) Marshal(v interface{}) ([]byte, error) {
	pb, ok := v.(proto.Message)
	if !ok {
		return nil, codec.ErrInvalidMessage
	}

	// looks not good, but allows to reuse underlining bytes
	buf := bufferPool.Get()
	pbuf := proto.NewBuffer(buf.Bytes())
	defer func() {
		bufferPool.Put(bytes.NewBuffer(pbuf.Bytes()))
	}()

	if err := pbuf.Marshal(pb); err != nil {
		return nil, err
	}

	return pbuf.Bytes(), nil
}

func (Marshaler) Unmarshal(data []byte, v interface{}) error {
	pb, ok := v.(proto.Message)
	if !ok {
		return codec.ErrInvalidMessage
	}

	return proto.Unmarshal(data, pb)
}

func (Marshaler) String() string {
	return "proto"
}
