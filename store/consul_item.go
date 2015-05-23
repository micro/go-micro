package store

type consulItem struct {
	key   string
	value []byte
}

func (c *consulItem) Key() string {
	return c.key
}

func (c *consulItem) Value() []byte {
	return c.value
}
