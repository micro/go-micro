// Package cloudflare is a store implementation backed by cloudflare workers kv
// Note that the cloudflare workers KV API is eventually consistent.
package cloudflare

import (
	"context"
	"log"
	"time"

	"github.com/cloudflare/cloudflare-go"
	"github.com/micro/go-micro/config/options"
	"github.com/micro/go-micro/store"
)

var namespaceUUID string

type workersKV struct {
	options.Options
	api *cloudflare.API
}

// New returns a cloudflare Store implementation.
// Options expects CF_API_TOKEN to a cloudflare API token scoped to Workers KV,
// CF_ACCOUNT_ID to contain a string with your cloudflare account ID and
// KV_NAMESPACE_ID to contain the namespace UUID for your KV storage.
func New(opts ...options.Option) (store.Store, error) {
	// Validate Options
	options := options.NewOptions(opts...)
	apiToken, ok := options.Values().Get("CF_API_TOKEN")
	if !ok {
		log.Fatal("Store: No CF_API_TOKEN passed as an option")
	}
	apiTokenString, ok := apiToken.(string)
	if !ok {
		log.Fatal("Store: Option CF_API_TOKEN contains a non-string")
	}
	accountID, ok := options.Values().Get("CF_ACCOUNT_ID")
	if !ok {
		log.Fatal("Store: No CF_ACCOUNT_ID passed as an option")
	}
	accountIDString, ok := accountID.(string)
	if !ok {
		log.Fatal("Store: Option CF_ACCOUNT_ID contains a non-string")
	}
	uuid, ok := options.Values().Get("KV_NAMESPACE_ID")
	if !ok {
		log.Fatal("Store: No KV_NAMESPACE_ID passed as an option")
	}
	namespaceUUID, ok = uuid.(string)
	if !ok {
		log.Fatal("Store: Option KV_NAMESPACE_ID contains a non-string")
	}

	// Create API client
	api, err := cloudflare.NewWithAPIToken(apiTokenString, cloudflare.UsingAccount(accountIDString))
	if err != nil {
		return nil, err
	}
	return &workersKV{
		Options: options,
		api:     api,
	}, nil
}

// In the cloudflare workers KV implemention, Sync() doesn't guarantee
// anything as the workers API is eventually consistent.
func (w *workersKV) Sync() ([]*store.Record, error) {
	response, err := w.api.ListWorkersKVs(context.Background(), namespaceUUID)
	if err != nil {
		return nil, err
	}
	var keys []string
	for _, r := range response.Result {
		keys = append(keys, r.Name)
	}
	return w.Read(keys...)
}

func (w *workersKV) Read(keys ...string) ([]*store.Record, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var records []*store.Record
	for _, k := range keys {
		v, err := w.api.ReadWorkersKV(ctx, namespaceUUID, k)
		if err != nil {
			return records, err
		}
		records = append(records, &store.Record{
			Key:   k,
			Value: v,
		})
	}
	return records, nil
}

func (w *workersKV) Write(records ...*store.Record) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for _, r := range records {
		if _, err := w.api.WriteWorkersKV(ctx, namespaceUUID, r.Key, r.Value); err != nil {
			return err
		}
	}
	return nil
}

func (w *workersKV) Delete(keys ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for _, k := range keys {
		if _, err := w.api.DeleteWorkersKV(ctx, namespaceUUID, k); err != nil {
			return err
		}
	}
	return nil
}
