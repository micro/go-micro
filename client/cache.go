package client

import (
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/micro/go-micro/v2/metadata"
)

// NewCache returns an initialised cache.
// TODO: Setup a go routine to expire records in the cache.
func NewCache() *Cache {
	return &Cache{
		values: make(map[string]interface{}),
	}
}

// Cache for responses
type Cache struct {
	values map[string]interface{}
	mutex  sync.Mutex
}

// Get a response from the cache
func (c *Cache) Get(ctx context.Context, req *Request) interface{} {
	md, _ := metadata.FromContext(ctx)
	ck := cacheKey{req, md}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	if val, ok := c.values[ck.Hash()]; ok {
		return val
	}

	return nil
}

// Set a response in the cache
func (c *Cache) Set(ctx context.Context, req *Request, rsp interface{}, expiry time.Duration) {
	md, _ := metadata.FromContext(ctx)
	ck := cacheKey{req, md}

	c.mutex.Lock()
	c.values[ck.Hash()] = rsp
	defer c.mutex.Unlock()
}

type cacheKey struct {
	Request  *Request
	Metadata metadata.Metadata
}

// Source: https://gobyexample.com/sha1-hashes
func (k *cacheKey) Hash() string {
	bytes, _ := json.Marshal(k)
	h := sha1.New()
	h.Write(bytes)
	return fmt.Sprintf("%x", h.Sum(nil))
}
