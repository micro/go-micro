package consul

import (
	"fmt"
	"log"
	"net"
	"os"
	"path"
	"sync"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/api/watch"
	"github.com/micro/go-micro/sync/leader"
)

type consulLeader struct {
	opts leader.Options
	c    *api.Client
}

type consulElected struct {
	c    *api.Client
	l    *api.Lock
	id   string
	key  string
	opts leader.ElectOptions

	mtx sync.RWMutex
	rv  <-chan struct{}
}

func (c *consulLeader) Elect(id string, opts ...leader.ElectOption) (leader.Elected, error) {
	var options leader.ElectOptions
	for _, o := range opts {
		o(&options)
	}

	key := path.Join("micro/leader", c.opts.Group)

	lc, err := c.c.LockOpts(&api.LockOptions{
		Key:   key,
		Value: []byte(id),
	})
	if err != nil {
		return nil, err
	}

	rv, err := lc.Lock(nil)
	if err != nil {
		return nil, err
	}

	return &consulElected{
		c:    c.c,
		key:  key,
		rv:   rv,
		id:   id,
		l:    lc,
		opts: options,
	}, nil
}

func (c *consulLeader) Follow() chan string {
	ch := make(chan string, 1)

	key := path.Join("/micro/leader", c.opts.Group)

	p, err := watch.Parse(map[string]interface{}{
		"type": "key",
		"key":  key,
	})
	if err != nil {
		return ch
	}
	p.Handler = func(idx uint64, raw interface{}) {
		if raw == nil {
			return // ignore
		}
		v, ok := raw.(*api.KVPair)
		if !ok || v == nil {
			return // ignore
		}
		ch <- string(v.Value)
	}

	go p.RunWithClientAndLogger(c.c, log.New(os.Stdout, "consul: ", log.Lshortfile))
	return ch
}

func (c *consulLeader) String() string {
	return "consul"
}

func (c *consulElected) Id() string {
	return c.id
}

func (c *consulElected) Reelect() error {
	rv, err := c.l.Lock(nil)
	if err != nil {
		return err
	}

	c.mtx.Lock()
	c.rv = rv
	c.mtx.Unlock()
	return nil
}

func (c *consulElected) Revoked() chan bool {
	ch := make(chan bool, 1)
	c.mtx.RLock()
	rv := c.rv
	c.mtx.RUnlock()

	go func() {
		<-rv
		ch <- true
		close(ch)
	}()

	return ch
}

func (c *consulElected) Resign() error {
	return c.l.Unlock()
}

func NewLeader(opts ...leader.Option) leader.Leader {
	options := leader.Options{
		Group: "default",
	}
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
			config.Address = fmt.Sprintf("%s:%s", addr, port)
		} else if err == nil {
			config.Address = fmt.Sprintf("%s:%s", addr, port)
		}
	}

	client, _ := api.NewClient(config)

	return &consulLeader{
		opts: options,
		c:    client,
	}
}
