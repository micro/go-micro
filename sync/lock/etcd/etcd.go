// Package etcd is an etcd implementation of lock
package etcd

import (
	"context"
	"errors"
	"log"
	"path"
	"strings"
	"sync"

	client "github.com/coreos/etcd/clientv3"
	cc "github.com/coreos/etcd/clientv3/concurrency"
	"github.com/micro/go-micro/sync/lock"
)

type etcdLock struct {
	opts   lock.Options
	path   string
	client *client.Client

	sync.Mutex
	locks map[string]*elock
}

type elock struct {
	s *cc.Session
	m *cc.Mutex
}

func (e *etcdLock) Acquire(id string, opts ...lock.AcquireOption) error {
	var options lock.AcquireOptions
	for _, o := range opts {
		o(&options)
	}

	// make path
	path := path.Join(e.path, strings.Replace(e.opts.Prefix+id, "/", "-", -1))

	var sopts []cc.SessionOption
	if options.TTL > 0 {
		sopts = append(sopts, cc.WithTTL(int(options.TTL.Seconds())))
	}

	s, err := cc.NewSession(e.client, sopts...)
	if err != nil {
		return err
	}

	m := cc.NewMutex(s, path)

	ctx, _ := context.WithCancel(context.Background())

	if err := m.Lock(ctx); err != nil {
		return err
	}

	e.Lock()
	e.locks[id] = &elock{
		s: s,
		m: m,
	}
	e.Unlock()
	return nil
}

func (e *etcdLock) Release(id string) error {
	e.Lock()
	defer e.Unlock()
	v, ok := e.locks[id]
	if !ok {
		return errors.New("lock not found")
	}
	err := v.m.Unlock(context.Background())
	delete(e.locks, id)
	return err
}

func (e *etcdLock) String() string {
	return "etcd"
}

func NewLock(opts ...lock.Option) lock.Lock {
	var options lock.Options
	for _, o := range opts {
		o(&options)
	}

	var endpoints []string

	for _, addr := range options.Nodes {
		if len(addr) > 0 {
			endpoints = append(endpoints, addr)
		}
	}

	if len(endpoints) == 0 {
		endpoints = []string{"http://127.0.0.1:2379"}
	}

	// TODO: parse addresses
	c, err := client.New(client.Config{
		Endpoints: endpoints,
	})
	if err != nil {
		log.Fatal(err)
	}

	return &etcdLock{
		path:   "/micro/lock",
		client: c,
		opts:   options,
		locks:  make(map[string]*elock),
	}
}
