package yaml

import "github.com/ghodss/yaml"

type yamlEncoder struct{}

func (y yamlEncoder) Encode(i interface{}) ([]byte, error) {
	return yaml.Marshal(i)
}

func (y yamlEncoder) Decode(bytes []byte, i interface{}) error {
	return yaml.Unmarshal(bytes, i)
}

func (y yamlEncoder) String() string {
	return "yaml"
}

func NewEncoder() *yamlEncoder {
	return &yamlEncoder{}
}
