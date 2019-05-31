// Package consul is a consul implemenation of lock
package consul

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/hashicorp/consul/api"
	lock "github.com/micro/go-micro/sync/lock"
)

type consulLock struct {
	sync.Mutex

	locks map[string]*api.Lock
	opts  lock.Options
	c     *api.Client
}

func (c *consulLock) Acquire(id string, opts ...lock.AcquireOption) error {
	var options lock.AcquireOptions
	for _, o := range opts {
		o(&options)
	}

	if options.Wait <= time.Duration(0) {
		options.Wait = api.DefaultLockWaitTime
	}

	ttl := fmt.Sprintf("%v", options.TTL)
	if options.TTL <= time.Duration(0) {
		ttl = api.DefaultLockSessionTTL
	}

	l, err := c.c.LockOpts(&api.LockOptions{
		Key:          c.opts.Prefix + id,
		LockWaitTime: options.Wait,
		SessionTTL:   ttl,
	})

	if err != nil {
		return err
	}

	_, err = l.Lock(nil)
	if err != nil {
		return err
	}

	c.Lock()
	c.locks[id] = l
	c.Unlock()

	return nil
}

func (c *consulLock) Release(id string) error {
	c.Lock()
	defer c.Unlock()
	l, ok := c.locks[id]
	if !ok {
		return errors.New("lock not found")
	}
	err := l.Unlock()
	delete(c.locks, id)
	return err
}

func (c *consulLock) String() string {
	return "consul"
}

func NewLock(opts ...lock.Option) lock.Lock {
	var options lock.Options
	for _, o := range opts {
		o(&options)
	}

	config := api.DefaultConfig()

	// set host
	// config.Host something
	// check if there are any addrs
	if len(options.Nodes) > 0 {
		addr, port, err := net.SplitHostPort(options.Nodes[0])
		if ae, ok := err.(*net.AddrError); ok && ae.Err == "missing port in address" {
			port = "8500"
			config.Address = fmt.Sprintf("%s:%s", options.Nodes[0], port)
		} else if err == nil {
			config.Address = fmt.Sprintf("%s:%s", addr, port)
		}
	}

	client, _ := api.NewClient(config)

	return &consulLock{
		locks: make(map[string]*api.Lock),
		opts:  options,
		c:     client,
	}
}
