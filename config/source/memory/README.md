# Memory Source

The memory source provides in-memory data as a source

## Memory Format

The expected data format is json

```json
data := []byte(`{
    "hosts": {
        "database": {
            "address": "10.0.0.1",
            "port": 3306
        },
        "cache": {
            "address": "10.0.0.2",
            "port": 6379
        }
    }
}`)
```

## New Source

Specify source with data

```go
memorySource := memory.NewSource(
	memory.WithJSON(data),
)
```

## Load Source

Load the source into config

```go
// Create new config
conf := config.NewConfig()

// Load memory source
conf.Load(memorySource)
```
