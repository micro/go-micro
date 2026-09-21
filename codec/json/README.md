# JSON and protobuf output

The JSON codec already detects modern `proto.Message` values and uses
`protojson`. Plain Go structs use `encoding/json`. Both default paths may omit
zero values. In generated Go structs, `omitempty` tags are produced by
`protoc-gen-go`; changing `protoc-gen-micro` cannot remove them.

For explicit zero values and original protobuf field names, configure a codec:

```go
factory := jsoncodec.NewCodecWithOptions(jsoncodec.Options{
    MarshalOptions: protojson.MarshalOptions{
        EmitUnpopulated: true,
        UseProtoNames: true,
    },
})
server.NewServer(server.Codec("application/json", factory))
client.NewClient(client.Codec("application/json", factory))
```

Imports: `jsoncodec "go-micro.dev/v6/codec/json"`,
`"google.golang.org/protobuf/encoding/protojson"`, and the Go Micro `client` and
`server` packages. Configure the side that serializes responses. For an HTTP
handler with a protobuf response, use the same options through
`jsoncodec.Marshaler{Options: options}.Marshal(response)` and write the returned
bytes with `Content-Type: application/json`.

Protobuf JSON encodes `int64` and `uint64` as **strings**, including zero, to avoid
JavaScript precision loss. An opt-in `Int64AsNumber: true` converts protobuf
64-bit integer values to numbers only within `[-(2^53-1), 2^53-1]` (nonnegative
for unsigned values). Larger values return an error, never a rounded number.
Lists, maps, nested messages, integer wrappers, and resolved `Any` messages are
handled recursively. Map keys and ordinary string fields remain strings.

Default codecs and HTTP wire formats remain unchanged in v6. Select these
options explicitly at the response boundary, especially for financial APIs
whose consumers expect numeric amounts. `MarshalOptions` and `UnmarshalOptions`
also expose the standard protobuf JSON settings, including custom type resolvers.
