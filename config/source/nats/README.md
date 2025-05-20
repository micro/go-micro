# Nats Source

The nats source reads config from nats key/values

## Nats Format

The nats source expects keys under the default bucket `default` default key `micro_config`

Values are expected to be json

```
nats kv put default micro_config '{"nats": {"address": "10.0.0.1", "port": 8488}}'
```

```
conf.Get("nats")
```

## New Source

Specify source with data

```go
natsSource := nats.NewSource(
	nats.WithUrl("127.0.0.1:4222"),
	nats.WithBucket("my_bucket"),
	nats.WithKey("my_key"),
)
```

## Load Source

Load the source into config

```go
// Create new config
conf := config.NewConfig()

// Load nats source
conf.Load(natsSource)
```

## Watch

```go
wh, _ := natsSource.Watch()

for {
	v, err := watcher.Next()
	if err != nil {
		log.Fatalf("err %v", err)
	}

	log.Infof("data %v", string(v.Data))
}
```
