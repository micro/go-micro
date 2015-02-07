package store

type EtcdItem struct {
	key   string
	value []byte
}

func (c *EtcdItem) Key() string {
	return c.key
}

func (c *EtcdItem) Value() []byte {
	return c.value
}
