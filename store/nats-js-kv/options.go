package natsjskv

import (
	"time"

	"github.com/nats-io/nats.go"
	"go-micro.dev/v5/store"
)

// store.Option.
type natsOptionsKey struct{}
type jsOptionsKey struct{}
type kvOptionsKey struct{}
type ttlOptionsKey struct{}
type memoryOptionsKey struct{}
type descriptionOptionsKey struct{}
type keyEncodeOptionsKey struct{}

// NatsOptions accepts nats.Options.
func NatsOptions(opts nats.Options) store.Option {
	return setStoreOption(natsOptionsKey{}, opts)
}

// JetStreamOptions accepts multiple nats.JSOpt.
func JetStreamOptions(opts ...nats.JSOpt) store.Option {
	return setStoreOption(jsOptionsKey{}, opts)
}

// KeyValueOptions accepts multiple nats.KeyValueConfig
// This will create buckets with the provided configs at initialization.
func KeyValueOptions(cfg ...*nats.KeyValueConfig) store.Option {
	return setStoreOption(kvOptionsKey{}, cfg)
}

// DefaultTTL sets the default TTL to use for new buckets
//
//	By default no TTL is set.
//
// TTL ON INDIVIDUAL WRITE CALLS IS NOT SUPPORTED, only bucket wide TTL.
// Either set a default TTL with this option or provide bucket specific options
//
//	with ObjectStoreOptions
func DefaultTTL(ttl time.Duration) store.Option {
	return setStoreOption(ttlOptionsKey{}, ttl)
}

// DefaultMemory sets the default storage type to memory only.
//
//	The default is file storage, persisting storage between service restarts.
//
// Be aware that the default storage location of NATS the /tmp dir is, and thus
//
//	won't persist reboots.
func DefaultMemory() store.Option {
	return setStoreOption(memoryOptionsKey{}, nats.MemoryStorage)
}

// DefaultDescription sets the default description to use when creating new
//
//	buckets. The default is "Store managed by go-micro"
func DefaultDescription(text string) store.Option {
	return setStoreOption(descriptionOptionsKey{}, text)
}

// EncodeKeys will "base32" encode the keys.
// This is to work around limited characters usable as keys for the natsjs kv store.
// See details here: https://docs.nats.io/nats-concepts/subjects#characters-allowed-for-subject-names
func EncodeKeys() store.Option {
	return setStoreOption(keyEncodeOptionsKey{}, "base32")
}

// DeleteBucket will use the key passed to Delete as a bucket (database) name,
//
//	and delete the bucket.
//
// This option should not be combined with the store.DeleteFrom option, as
//
//	that will overwrite the delete action.
func DeleteBucket() store.DeleteOption {
	return func(d *store.DeleteOptions) {
		d.Table = "DELETE_BUCKET"
	}
}
