// Package redis is a redis implemenation of lock
package redis

import (
	"errors"
	"sync"
	"time"

	"github.com/go-redsync/redsync"
	"github.com/micro/go-micro/sync/lock"
)

type redisLock struct {
	sync.Mutex

	locks map[string]*redsync.Mutex
	opts  lock.Options
	c     *redsync.Redsync
}

func (r *redisLock) Acquire(id string, opts ...lock.AcquireOption) error {
	var options lock.AcquireOptions
	for _, o := range opts {
		o(&options)
	}

	var ropts []redsync.Option

	if options.Wait > time.Duration(0) {
		ropts = append(ropts, redsync.SetRetryDelay(options.Wait))
		ropts = append(ropts, redsync.SetTries(1))
	}

	if options.TTL > time.Duration(0) {
		ropts = append(ropts, redsync.SetExpiry(options.TTL))
	}

	m := r.c.NewMutex(r.opts.Prefix+id, ropts...)
	err := m.Lock()
	if err != nil {
		return err
	}

	r.Lock()
	r.locks[id] = m
	r.Unlock()

	return nil
}

func (r *redisLock) Release(id string) error {
	r.Lock()
	defer r.Unlock()
	m, ok := r.locks[id]
	if !ok {
		return errors.New("lock not found")
	}

	unlocked := m.Unlock()
	delete(r.locks, id)

	if !unlocked {
		return errors.New("lock not unlocked")
	}

	return nil
}

func (r *redisLock) String() string {
	return "redis"
}

func NewLock(opts ...lock.Option) lock.Lock {
	var options lock.Options
	for _, o := range opts {
		o(&options)
	}

	nodes := options.Nodes

	if len(nodes) == 0 {
		nodes = []string{"127.0.0.1:6379"}
	}

	rpool := redsync.New([]redsync.Pool{&pool{
		addrs: nodes,
	}})

	return &redisLock{
		locks: make(map[string]*redsync.Mutex),
		opts:  options,
		c:     rpool,
	}
}
