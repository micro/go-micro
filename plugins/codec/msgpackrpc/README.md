# MsgPack RPC Codec

## Usage

Import the codec and set within the client/server
```go
package main

import (
    "github.com/asim/go-micro/plugins/codec/msgpackrpc"
    "github.com/asim/go-micro/v3"
    "github.com/micro/go-micro/client"
    "github.com/micro/go-micro/server"
)

func main() {
    client := client.NewClient(
        client.Codec("application/msgpack", msgpackrpc.NewCodec),
        client.ContentType("application/msgpack"),
    )

    server := server.NewServer(
        server.Codec("application/msgpack", msgpackrpc.NewCodec),
    )

    service := micro.NewService(
        micro.Client(client),
        micro.Server(server),
    )

    // ...
}
```

## Generating Request/Response types

The msgpack codec is much like protobuf. It expects the request/response types to conform to a specific interface. Usually this 
means defining some IDL and generating the required types. 

Learn how to do that at [github.com/tinylib/msgp](https://github.com/tinylib/msgp)

