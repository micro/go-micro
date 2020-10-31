package json

import (
	"errors"
	"time"

	"github.com/asim/nitro/v3/codec"
	"github.com/asim/nitro/v3/codec/json"
	"github.com/asim/nitro/v3/config/reader"
	"github.com/asim/nitro/v3/config/source"
	"github.com/imdario/mergo"
)

type jsonReader struct {
	opts reader.Options
	json codec.Marshaler
}

func (j *jsonReader) Merge(changes ...*source.ChangeSet) (*source.ChangeSet, error) {
	var merged map[string]interface{}

	for _, m := range changes {
		if m == nil {
			continue
		}

		if len(m.Data) == 0 {
			continue
		}

		codec := j.json

		var data map[string]interface{}
		if err := codec.Unmarshal(m.Data, &data); err != nil {
			return nil, err
		}
		if err := mergo.Map(&merged, data, mergo.WithOverride); err != nil {
			return nil, err
		}
	}

	b, err := j.json.Marshal(merged)
	if err != nil {
		return nil, err
	}

	cs := &source.ChangeSet{
		Timestamp: time.Now(),
		Data:      b,
		Source:    "json",
		Format:    j.json.String(),
	}
	cs.Checksum = cs.Sum()

	return cs, nil
}

func (j *jsonReader) Values(ch *source.ChangeSet) (reader.Values, error) {
	if ch == nil {
		return nil, errors.New("changeset is nil")
	}
	if ch.Format != "json" {
		return nil, errors.New("unsupported format")
	}
	return newValues(ch, j.opts)
}

func (j *jsonReader) String() string {
	return "json"
}

// NewReader creates a json reader
func NewReader(opts ...reader.Option) reader.Reader {
	options := reader.NewOptions(opts...)
	return &jsonReader{
		json: new(json.Marshaler),
		opts: options,
	}
}
