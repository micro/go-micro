package store

type MemcacheItem struct {
	key   string
	value []byte
}

func (m *MemcacheItem) Key() string {
	return m.key
}

func (m *MemcacheItem) Value() []byte {
	return m.value
}
