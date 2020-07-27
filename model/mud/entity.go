package mud

import (
	"github.com/google/uuid"
	"github.com/micro/go-micro/v3/codec"
	"github.com/micro/go-micro/v3/model"
)

type mudEntity struct {
	id         string
	name       string
	value      interface{}
	codec      codec.Marshaler
	attributes map[string]interface{}
}

func (m *mudEntity) Attributes() map[string]interface{} {
	return m.attributes
}

func (m *mudEntity) Id() string {
	return m.id
}

func (m *mudEntity) Name() string {
	return m.name
}

func (m *mudEntity) Value() interface{} {
	return m.value
}

func (m *mudEntity) Read(v interface{}) error {
	switch m.value.(type) {
	case []byte:
		b := m.value.([]byte)
		return m.codec.Unmarshal(b, v)
	default:
		v = m.value
	}
	return nil
}

func newEntity(name string, value interface{}, codec codec.Marshaler) model.Entity {
	return &mudEntity{
		id:         uuid.New().String(),
		name:       name,
		value:      value,
		codec:      codec,
		attributes: make(map[string]interface{}),
	}
}
