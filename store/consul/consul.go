package consul

// This is a hack

import (
	"github.com/myodc/go-micro/store"
)

func NewStore(addrs []string, opt ...store.Option) store.Store {
	return store.NewStore(addrs, opt...)
}
