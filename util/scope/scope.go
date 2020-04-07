package scope

import (
	"fmt"

	"github.com/micro/go-micro/v2/store"
)

// Scope extends the store, applying a prefix to each request
type Scope struct {
	store.Store
	prefix string
}

// NewScope returns an initialised scope
func NewScope(s store.Store, prefix string) Scope {
	return Scope{Store: s, prefix: prefix}
}

func (s *Scope) Options() store.Options {
	o := s.Store.Options()
	o.Table = s.prefix
	return o
}

func (s *Scope) Read(key string, opts ...store.ReadOption) ([]*store.Record, error) {
	key = fmt.Sprintf("%v/%v", s.prefix, key)
	return s.Store.Read(key, opts...)
}

func (s *Scope) Write(r *store.Record, opts ...store.WriteOption) error {
	r.Key = fmt.Sprintf("%v/%v", s.prefix, r.Key)
	return s.Store.Write(r, opts...)
}

func (s *Scope) Delete(key string, opts ...store.DeleteOption) error {
	key = fmt.Sprintf("%v/%v", s.prefix, key)
	return s.Store.Delete(key, opts...)
}

func (s *Scope) List(opts ...store.ListOption) ([]string, error) {
	var lops store.ListOptions
	for _, o := range opts {
		o(&lops)
	}

	key := fmt.Sprintf("%v/%v", s.prefix, lops.Prefix)
	opts = append(opts, store.ListPrefix(key))

	return s.Store.List(opts...)
}
