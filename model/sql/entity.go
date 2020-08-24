package sql

import (
	"github.com/google/uuid"
	"github.com/micro/go-micro/v3/codec"
	"github.com/micro/go-micro/v3/model"
)

type sqlEntity struct {
	id         string
	name       string
	value      interface{}
	codec      codec.Marshaler
	attributes map[string]interface{}
}

func (m *sqlEntity) Attributes() map[string]interface{} {
	return m.attributes
}

func (m *sqlEntity) Id() string {
	return m.id
}

func (m *sqlEntity) Name() string {
	return m.name
}

func (m *sqlEntity) Value() interface{} {
	return m.value
}

func newEntity(name string, value interface{}, codec codec.Marshaler) model.Entity {
	return &sqlEntity{
		id:         uuid.New().String(),
		name:       name,
		value:      value,
		codec:      codec,
		attributes: make(map[string]interface{}),
	}
}
