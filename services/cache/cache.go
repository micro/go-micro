package cache

import (
	"go-micro.dev/v4/api/client"
)

type Cache interface {
	Decrement(*DecrementRequest) (*DecrementResponse, error)
	Delete(*DeleteRequest) (*DeleteResponse, error)
	Get(*GetRequest) (*GetResponse, error)
	Increment(*IncrementRequest) (*IncrementResponse, error)
	Set(*SetRequest) (*SetResponse, error)
}

func NewCacheService(token string) *CacheService {
	return &CacheService{
		client: client.NewClient(&client.Options{
			Token: token,
		}),
	}
}

type CacheService struct {
	client *client.Client
}

// Decrement a value (if it's a number). If key not found it is equivalent to set.
func (t *CacheService) Decrement(request *DecrementRequest) (*DecrementResponse, error) {

	rsp := &DecrementResponse{}
	return rsp, t.client.Call("cache", "Decrement", request, rsp)

}

// Delete a value from the cache. If key not found a success response is returned.
func (t *CacheService) Delete(request *DeleteRequest) (*DeleteResponse, error) {

	rsp := &DeleteResponse{}
	return rsp, t.client.Call("cache", "Delete", request, rsp)

}

// Get an item from the cache by key. If key is not found, an empty response is returned.
func (t *CacheService) Get(request *GetRequest) (*GetResponse, error) {

	rsp := &GetResponse{}
	return rsp, t.client.Call("cache", "Get", request, rsp)

}

// Increment a value (if it's a number). If key not found it is equivalent to set.
func (t *CacheService) Increment(request *IncrementRequest) (*IncrementResponse, error) {

	rsp := &IncrementResponse{}
	return rsp, t.client.Call("cache", "Increment", request, rsp)

}

// Set an item in the cache. Overwrites any existing value already set.
func (t *CacheService) Set(request *SetRequest) (*SetResponse, error) {

	rsp := &SetResponse{}
	return rsp, t.client.Call("cache", "Set", request, rsp)

}

type DecrementRequest struct {
	// The key to decrement
	Key string `json:"key"`
	// The amount to decrement the value by
	Value int64 `json:"value,string"`
}

type DecrementResponse struct {
	// The key decremented
	Key string `json:"key"`
	// The new value
	Value int64 `json:"value,string"`
}

type DeleteRequest struct {
	// The key to delete
	Key string `json:"key"`
}

type DeleteResponse struct {
	// Returns "ok" if successful
	Status string `json:"status"`
}

type GetRequest struct {
	// The key to retrieve
	Key string `json:"key"`
}

type GetResponse struct {
	// The key
	Key string `json:"key"`
	// Time to live in seconds
	Ttl int64 `json:"ttl,string"`
	// The value
	Value string `json:"value"`
}

type IncrementRequest struct {
	// The key to increment
	Key string `json:"key"`
	// The amount to increment the value by
	Value int64 `json:"value,string"`
}

type IncrementResponse struct {
	// The key incremented
	Key string `json:"key"`
	// The new value
	Value int64 `json:"value,string"`
}

type SetRequest struct {
	// The key to update
	Key string `json:"key"`
	// Time to live in seconds
	Ttl int64 `json:"ttl,string"`
	// The value to set
	Value string `json:"value"`
}

type SetResponse struct {
	// Returns "ok" if successful
	Status string `json:"status"`
}
