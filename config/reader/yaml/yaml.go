package yaml

import (
	"errors"
	"github.com/asim/go-micro/v3/config/encoder"
	"github.com/asim/go-micro/v3/config/encoder/yaml"
	"github.com/asim/go-micro/v3/config/reader"
	"github.com/asim/go-micro/v3/config/source"
	"github.com/imdario/mergo"
	"time"
)

type yamlReader struct {
	opts reader.Options
	yaml encoder.Encoder
}

func (y *yamlReader) Merge(sets ...*source.ChangeSet) (*source.ChangeSet, error) {
	var merged map[string]interface{}

	for _, v := range sets {
		if v == nil {
			continue
		}

		if len(v.Data) == 0 {
			continue
		}

		e, ok := y.opts.Encoding[v.Format]
		if !ok {
			e = y.yaml
		}

		var data map[string]interface{}
		if err := e.Decode(v.Data, &data); err != nil {
			return nil, err
		}

		if err := mergo.Map(&merged, data, mergo.WithOverride); err != nil {
			return nil, err
		}
	}

	b, err := y.yaml.Encode(merged)
	if err != nil {
		return nil, err
	}

	cs := &source.ChangeSet{
		Data:      b,
		Format:    y.yaml.String(),
		Source:    "yaml",
		Timestamp: time.Now(),
	}
	cs.Checksum = cs.Sum()

	return cs, nil
}

func (y *yamlReader) Values(sets *source.ChangeSet) (reader.Values, error) {
	if sets == nil {
		return nil, errors.New("changeset is nil")
	}
	if sets.Format != "yaml" {
		return nil, errors.New("unsupported format")
	}

	return newValues(sets)
}

func (y *yamlReader) String() string {
	return "yaml"
}

// NewReader creates a yaml reader
func NewReader(opts ...reader.Option) reader.Reader {
	options := reader.NewOptions(opts...)
	return &yamlReader{
		opts: options,
		yaml: yaml.NewEncoder(),
	}
}
