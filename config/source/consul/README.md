# Consul Source

The consul source reads config from consul key/values

## Consul Format

The consul source expects keys under the default prefix `/micro/config`

Values are expected to be json

```
// set database
consul kv put micro/config/database '{"address": "10.0.0.1", "port": 3306}'
// set cache
consul kv put micro/config/cache '{"address": "10.0.0.2", "port": 6379}'
```

Keys are split on `/` so access becomes

```
conf.Get("micro", "config", "database")
```

## New Source

Specify source with data

```go
consulSource := consul.NewSource(
	// optionally specify consul address; default to localhost:8500
	consul.WithAddress("10.0.0.10:8500"),
	// optionally specify prefix; defaults to /micro/config
	consul.WithPrefix("/my/prefix"),
  // optionally strip the provided prefix from the keys, defaults to false
  consul.StripPrefix(true),
)
```

## Load Source

Load the source into config

```go
// Create new config
conf := config.NewConfig()

// Load consul source
conf.Load(consulSource)
```
