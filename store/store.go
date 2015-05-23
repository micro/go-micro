package store

type Store interface {
	Get(string) (*Item, error)
	Del(string) error
	Put(*Item) error
}

type Item struct {
	Key   string
	Value []byte
}

type options struct{}

type Option func(*options)

var (
	DefaultStore = newConsulStore([]string{})
)

func NewStore(addrs []string, opt ...Option) Store {
	return newConsulStore(addrs, opt...)
}

func Get(key string) (*Item, error) {
	return DefaultStore.Get(key)
}

func Del(key string) error {
	return DefaultStore.Del(key)
}

func Put(item *Item) error {
	return DefaultStore.Put(item)
}
