package store

type Store interface {
	Get(string) (Item, error)
	Del(string) error
	Put(Item) error
	NewItem(string, []byte) Item
}

var (
	DefaultStore = NewConsulStore()
)

func Get(key string) (Item, error) {
	return DefaultStore.Get(key)
}

func Del(key string) error {
	return DefaultStore.Del(key)
}

func Put(item Item) error {
	return DefaultStore.Put(item)
}

func NewItem(key string, value []byte) Item {
	return DefaultStore.NewItem(key, value)
}
