# NATS JetStream Key Value Store Plugin

This plugin uses the NATS JetStream [KeyValue Store](https://docs.nats.io/nats-concepts/jetstream/key-value-store) to implement the Go-Micro store interface.

You can use this plugin like any other store plugin. 
To start a local NATS JetStream server run `nats-server -js`.

To manually create a new storage object call:

```go
natsjskv.NewStore(opts ...store.Option)
```

The Go-Micro store interface uses databases and tables to store keys. These translate
to buckets (key value stores) and key prefixes. If no database (bucket name) is provided, "default" will be used.

You can call `Write` with any arbitrary database name, and if a bucket with that name does not exist yet,
it will be automatically created.

If a table name is provided, it will use it to prefix the key as `<table>_<key>`.

To delete a bucket, and all the key/value pairs in it, pass the `DeleteBucket` option to the `Delete`
method, then they key name will be interpreted as a bucket name, and the bucket will be deleted.

Next to the default store options, a few NATS specific options are available:


```go
// NatsOptions accepts nats.Options
NatsOptions(opts nats.Options)

// JetStreamOptions accepts multiple nats.JSOpt
JetStreamOptions(opts ...nats.JSOpt)

// KeyValueOptions accepts multiple nats.KeyValueConfig
// This will create buckets with the provided configs at initialization.
//
// type KeyValueConfig struct {
//    Bucket       string
//   Description  string
//   MaxValueSize int32
//   History      uint8
//   TTL          time.Duration
//   MaxBytes     int64
//   Storage      StorageType
//   Replicas     int
//   Placement    *Placement
//   RePublish    *RePublish
//   Mirror       *StreamSource
//   Sources      []*StreamSource
}
KeyValueOptions(cfg ...*nats.KeyValueConfig)

// DefaultTTL sets the default TTL to use for new buckets
//  By default no TTL is set.
//
// TTL ON INDIVIDUAL WRITE CALLS IS NOT SUPPORTED, only bucket wide TTL.
// Either set a default TTL with this option or provide bucket specific options
//  with ObjectStoreOptions
DefaultTTL(ttl time.Duration)

// DefaultMemory sets the default storage type to memory only.
//
//  The default is file storage, persisting storage between service restarts.
// Be aware that the default storage location of NATS the /tmp dir is, and thus
//  won't persist reboots.
DefaultMemory()

// DefaultDescription sets the default description to use when creating new
//  buckets. The default is "Store managed by go-micro"
DefaultDescription(text string)

// DeleteBucket will use the key passed to Delete as a bucket (database) name,
//  and delete the bucket.
// This option should not be combined with the store.DeleteFrom option, as
//  that will overwrite the delete action.
DeleteBucket()
```

