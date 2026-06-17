package store

// Scope returns a Store that confines every operation to the given
// database and table of s, without mutating s. It is the safe way to give
// each component — a service, an agent, a flow — its own table over a
// shared backend: unlike Init(Table(...)), which changes process-global
// options and so races between co-located components, a scoped handle
// injects the database/table on each call.
//
// An empty database or table falls through to the underlying store's
// default for that field.
//
//	st := store.Scope(store.DefaultStore, "agent", "task-mgr")
//	st.Write(&store.Record{Key: "history", Value: data}) // -> agent/task-mgr
func Scope(s Store, database, table string) Store {
	return &scopedStore{Store: s, database: database, table: table}
}

// scopedStore applies a fixed database/table to every data operation,
// delegating everything else to the embedded Store.
type scopedStore struct {
	Store
	database string
	table    string
}

func (s *scopedStore) Read(key string, opts ...ReadOption) ([]*Record, error) {
	return s.Store.Read(key, append([]ReadOption{ReadFrom(s.database, s.table)}, opts...)...)
}

func (s *scopedStore) Write(r *Record, opts ...WriteOption) error {
	return s.Store.Write(r, append([]WriteOption{WriteTo(s.database, s.table)}, opts...)...)
}

func (s *scopedStore) Delete(key string, opts ...DeleteOption) error {
	return s.Store.Delete(key, append([]DeleteOption{DeleteFrom(s.database, s.table)}, opts...)...)
}

func (s *scopedStore) List(opts ...ListOption) ([]string, error) {
	return s.Store.List(append([]ListOption{ListFrom(s.database, s.table)}, opts...)...)
}
