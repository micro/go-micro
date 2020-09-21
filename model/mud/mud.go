// Package mud is the micro data model implementation
package mud

import (
	"github.com/micro/go-micro/v3/codec/json"
	"github.com/micro/go-micro/v3/model"
	"github.com/micro/go-micro/v3/store"
	"github.com/micro/go-micro/v3/store/memory"
	memsync "github.com/micro/go-micro/v3/sync/memory"
)

type mudModel struct {
	options model.Options
}

func (m *mudModel) Init(opts ...model.Option) error {
	for _, o := range opts {
		o(&m.options)
	}
	return nil
}

func (m *mudModel) NewEntity(name string, value interface{}) model.Entity {
	// TODO: potentially pluralise name for tables
	return newEntity(name, value, m.options.Codec)
}

func (m *mudModel) Create(e model.Entity) error {
	// lock on the name of entity
	if err := m.options.Sync.Lock(e.Name()); err != nil {
		return err
	}
	// TODO: deal with the error
	defer m.options.Sync.Unlock(e.Name())

	// TODO: potentially add encode to entity?
	v, err := m.options.Codec.Marshal(e.Value())
	if err != nil {
		return err
	}

	// TODO: include metadata and set database
	return m.options.Store.Write(&store.Record{
		Key:   e.Id(),
		Value: v,
	}, store.WriteTo(m.options.Database, e.Name()))
}

func (m *mudModel) Read(opts ...model.ReadOption) ([]model.Entity, error) {
	var options model.ReadOptions
	for _, o := range opts {
		o(&options)
	}
	// TODO: implement the options that allow querying
	return nil, nil
}

func (m *mudModel) Update(e model.Entity) error {
	// TODO: read out the record first, update the fields and store

	// lock on the name of entity
	if err := m.options.Sync.Lock(e.Name()); err != nil {
		return err
	}
	// TODO: deal with the error
	defer m.options.Sync.Unlock(e.Name())

	// TODO: potentially add encode to entity?
	v, err := m.options.Codec.Marshal(e.Value())
	if err != nil {
		return err
	}

	// TODO: include metadata and set database
	return m.options.Store.Write(&store.Record{
		Key:   e.Id(),
		Value: v,
	}, store.WriteTo(m.options.Database, e.Name()))
}

func (m *mudModel) Delete(opts ...model.DeleteOption) error {
	var options model.DeleteOptions
	for _, o := range opts {
		o(&options)
	}
	// TODO: implement the options that allow deleting
	return nil
}

func (m *mudModel) String() string {
	return "mud"
}

func NewModel(opts ...model.Option) model.Model {
	options := model.Options{
		Codec: new(json.Marshaler),
		Sync:  memsync.NewSync(),
		Store: memory.NewStore(),
	}

	return &mudModel{
		options: options,
	}
}
