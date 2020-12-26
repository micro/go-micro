# Runtimevar Source

The runtimevar source is a source for the [Go Cloud Development Kit](https://github.com/google/go-cloud) runtimevar package.

This package takes a [runtimevar.Variable](https://godoc.org/gocloud.dev/runtimevar/#Variable)
and then allows you to use it as a backend source. When constructing your
`runtimevar.Variable`, use the `gocloud.dev/runtimevar.BytesDecoder` decoder to
allow your [Snapshot](https://godoc.org/gocloud.dev/runtimevar#Snapshot) value
to be `[]byte`. We then use the built in go-config encoder to decode the value. This defaults to json.

## New Source

Specify a runtimevar source with the Go CDK runtimevar.Variable. It will panic if not specified.

```go
// See https://godoc.org/gocloud.dev/runtimevar for examples on how to create
// a gocloud.dev/runtimevar.Varible. Use a BytesDecoder.
srv := runtimevar.NewSource(
	runtimevar.WithVariable(v),
)
```

## Config Format

To load different runtimevar formats e.g yaml, toml, xml you must specify an encoder.

```
e := toml.NewEncoder()

src := runtimevar.NewSource(
        runtimevar.WithVariable(v),
	source.WithEncoder(e),
)
```

## Load Source

Load the source into config

```go
// Create new config
conf := config.NewConfig()

// Load runtimevar source
conf.Load(src)
```

