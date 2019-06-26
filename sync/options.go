package sync

import (
	"github.com/micro/go-micro/data/store"
	"github.com/micro/go-micro/sync/leader"
	"github.com/micro/go-micro/sync/lock"
	"github.com/micro/go-micro/sync/time"
)

// WithLeader sets the leader election implementation opton
func WithLeader(l leader.Leader) Option {
	return func(o *Options) {
		o.Leader = l
	}
}

// WithLock sets the locking implementation option
func WithLock(l lock.Lock) Option {
	return func(o *Options) {
		o.Lock = l
	}
}

// WithStore sets the store implementation option
func WithStore(s store.Store) Option {
	return func(o *Options) {
		o.Store = s
	}
}

// WithTime sets the time implementation option
func WithTime(t time.Time) Option {
	return func(o *Options) {
		o.Time = t
	}
}
