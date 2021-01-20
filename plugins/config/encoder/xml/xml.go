package xml

import (
	"encoding/xml"

	"github.com/micro/go-micro/v2/config/encoder"
)

type xmlEncoder struct{}

func (x xmlEncoder) Encode(v interface{}) ([]byte, error) {
	return xml.Marshal(v)
}

func (x xmlEncoder) Decode(d []byte, v interface{}) error {
	return xml.Unmarshal(d, v)
}

func (x xmlEncoder) String() string {
	return "xml"
}

func NewEncoder() encoder.Encoder {
	return xmlEncoder{}
}
