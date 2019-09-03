package cue

import (
	"cuelang.org/go/cue"
	"github.com/ghodss/yaml"
	"github.com/micro/go-micro/config/encoder"
)

type cueEncoder struct{}

func (c cueEncoder) Encode(v interface{}) ([]byte, error) {
	return yaml.Marshal(v)
}

func (c cueEncoder) Decode(d []byte, v interface{}) error {
	var r cue.Runtime
	instance, err := r.Compile("config", d)
	if err != nil {
		return err
	}

	j, err := instance.Value().MarshalJSON()
	if err != nil {
		return err
	}
	return yaml.Unmarshal(j, v)
}

func (c cueEncoder) String() string {
	return "cue"
}

// NewEncoder : create new cueEncoder
func NewEncoder() encoder.Encoder {
	return cueEncoder{}
}
