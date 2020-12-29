// Package etcd is an etcd implementation of lock
package etcd

import (
	"context"
	"errors"
	"log"
	"path"
	"strings"
	gosync "sync"

	client "github.com/coreos/etcd/clientv3"
	cc "github.com/coreos/etcd/clientv3/concurrency"
	"github.com/micro/go-micro/v2/sync"
)

type etcdSync struct {
	options sync.Options
	path    string
	client  *client.Client

	mtx   gosync.Mutex
	locks map[string]*etcdLock
}

type etcdLock struct {
	s *cc.Session
	m *cc.Mutex
}

type etcdLeader struct {
	opts sync.LeaderOptions
	s    *cc.Session
	e    *cc.Election
	id   string
}

func (e *etcdSync) Leader(id string, opts ...sync.LeaderOption) (sync.Leader, error) {
	var options sync.LeaderOptions
	for _, o := range opts {
		o(&options)
	}

	// make path
	path := path.Join(e.path, strings.Replace(e.options.Prefix+id, "/", "-", -1))

	s, err := cc.NewSession(e.client)
	if err != nil {
		return nil, err
	}

	l := cc.NewElection(s, path)

	if err := l.Campaign(context.TODO(), id); err != nil {
		return nil, err
	}

	return &etcdLeader{
		opts: options,
		e:    l,
		id:   id,
	}, nil
}

func (e *etcdLeader) Status() chan bool {
	ch := make(chan bool, 1)
	ech := e.e.Observe(context.Background())

	go func() {
		for r := range ech {
			if string(r.Kvs[0].Value) != e.id {
				ch <- true
				close(ch)
				return
			}
		}
	}()

	return ch
}

func (e *etcdLeader) Resign() error {
	return e.e.Resign(context.Background())
}

func (e *etcdSync) Init(opts ...sync.Option) error {
	for _, o := range opts {
		o(&e.options)
	}
	return nil
}

func (e *etcdSync) Options() sync.Options {
	return e.options
}

func (e *etcdSync) Lock(id string, opts ...sync.LockOption) error {
	var options sync.LockOptions
	for _, o := range opts {
		o(&options)
	}

	// make path
	path := path.Join(e.path, strings.Replace(e.options.Prefix+id, "/", "-", -1))

	var sopts []cc.SessionOption
	if options.TTL > 0 {
		sopts = append(sopts, cc.WithTTL(int(options.TTL.Seconds())))
	}

	s, err := cc.NewSession(e.client, sopts...)
	if err != nil {
		return err
	}

	m := cc.NewMutex(s, path)

	if err := m.Lock(context.TODO()); err != nil {
		return err
	}

	e.mtx.Lock()
	e.locks[id] = &etcdLock{
		s: s,
		m: m,
	}
	e.mtx.Unlock()
	return nil
}

func (e *etcdSync) Unlock(id string) error {
	e.mtx.Lock()
	defer e.mtx.Unlock()
	v, ok := e.locks[id]
	if !ok {
		return errors.New("lock not found")
	}
	err := v.m.Unlock(context.Background())
	delete(e.locks, id)
	return err
}

func (e *etcdSync) String() string {
	return "etcd"
}

func NewSync(opts ...sync.Option) sync.Sync {
	var options sync.Options
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

	return &etcdSync{
		path:    "/micro/sync",
		client:  c,
		options: options,
		locks:   make(map[string]*etcdLock),
	}
}
