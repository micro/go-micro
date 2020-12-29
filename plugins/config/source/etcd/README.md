# Etcd Source

The etcd source reads config from etcd key/values

This source supports etcd version 3 and beyond.

## Etcd Format

The etcd source expects keys under the default prefix `/micro/config` (prefix can be changed)

Values are expected to be JSON

```
// set database
etcdctl put /micro/config/database '{"address": "10.0.0.1", "port": 3306}'
// set cache
etcdctl put /micro/config/cache '{"address": "10.0.0.2", "port": 6379}'
```

Keys are split on `/` so access becomes

```
conf.Get("micro", "config", "database")
```

## New Source

Specify source with data

```go
etcdSource := etcd.NewSource(
	// optionally specify etcd address; default to localhost:8500
	etcd.WithAddress("10.0.0.10:8500"),
	// optionally specify prefix; defaults to /micro/config
	etcd.WithPrefix("/my/prefix"),
	// optionally strip the provided prefix from the keys, defaults to false
	etcd.StripPrefix(true),
)
```

## Load Source

Load the source into config

```go
// Create new config
conf := config.NewConfig()

// Load file source
conf.Load(etcdSource)
```
