package json

import (
	"encoding/json"
)

type Marshaler struct{}

func (j Marshaler) Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func (j Marshaler) Unmarshal(d []byte, v interface{}) error {
	return json.Unmarshal(d, v)
}

func (j Marshaler) String() string {
	return "json"
}
