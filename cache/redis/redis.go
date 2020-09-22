package redis

import (
	"log"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/micro/go-micro/v3/cache"
	redisCluster "github.com/mna/redisc"
)

type redisCache struct {
	options cache.Options
	client  redisCluster.Cluster
}

func (r *redisCache) Init(opts ...cache.Option) error {
	for _, o := range opts {
		o(&r.options)
	}
	return nil
}

func (m *redisCache) Get(key string) (interface{}, error) {
	conn := m.client.Get()
	defer conn.Close()

	return redis.String(conn.Do("GET", key))
}

func (m *redisCache) Set(key string, val interface{}) error {
	conn := m.client.Get()
	defer conn.Close()

	_, err := conn.Do("SET", key, val)
	return err
}

func (m *redisCache) Delete(key string) error {
	conn := m.client.Get()
	defer conn.Close()

	_, err := conn.Do("DEL", key)
	return err
}

func (m *redisCache) String() string {
	return "redis"
}

// NewCache returns a new redis Cache
func NewCache(opts ...cache.Option) cache.Cache {
	var options cache.Options
	for _, o := range opts {
		o(&options)
	}

	// get and set the nodes
	nodes := options.Nodes
	if len(nodes) == 0 {
		nodes = []string{"127.0.0.1:6379", "127.0.0.1:6380", "127.0.0.1:6381"}
	}

	cluster := redisCluster.Cluster{
		StartupNodes: nodes,
		DialOptions:  []redis.DialOption{redis.DialConnectTimeout(5 * time.Second)},
		CreatePool:   createPool,
	}

	if err := cluster.Refresh(); err != nil {
		log.Fatalf("NewCache cluster.Refresh error(%v)", err)
	}

	return &redisCache{
		options: options,
		client:  cluster,
	}
}

func createPool(addr string, opts ...redis.DialOption) (*redis.Pool, error) {
	return &redis.Pool{
		MaxIdle:     5,
		MaxActive:   10,
		IdleTimeout: time.Millisecond * 100,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", addr, opts...)
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}, nil
}
