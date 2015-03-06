package store

type MemoryItem struct {
	key   string
	value []byte
}

func (m *MemoryItem) Key() string {
	return m.key
}

func (m *MemoryItem) Value() []byte {
	return m.value
}
