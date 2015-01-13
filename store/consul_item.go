package store

type ConsulItem struct {
	key   string
	value []byte
}

func (c *ConsulItem) Key() string {
	return c.key
}

func (c *ConsulItem) Value() []byte {
	return c.value
}
