# Flag Source

The flag source reads config from flags

## Format

We expect the use of the `flag` package. Upper case flags will be lower cased. Dashes will be used as delimiters.

### Example

```
dbAddress := flag.String("database_address", "127.0.0.1", "the db address")
dbPort := flag.Int("database_port", 3306, "the db port)
```

Becomes

```json
{
    "database": {
        "address": "127.0.0.1",
        "port": 3306
    }
}
```

## New Source

```go
flagSource := flag.NewSource(
	// optionally enable reading of unset flags and their default
	// values into config, defaults to false
	IncludeUnset(true)
)
```

## Load Source

Load the source into config

```go
// Create new config
conf := config.NewConfig()

// Load flag source
conf.Load(flagSource)
```
