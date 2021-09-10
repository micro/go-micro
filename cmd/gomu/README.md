# Gomu

Gomu is a helper tool for developing [Go Micro][1] projects.

## Getting Started

[Download][2] and install **Go**. Version `1.16` or higher is required.

Installation is done by using the [`go install`][3] command.

```bash
go install github.com/asim/go-micro/cmd/gomu@latest
```

Let's create a new service using the `new` command.

```bash
gomu new service helloworld
```

Follow the on-screen instructions. Next, we can run the program.

```bash
cd helloworld
make proto tidy
gomu run
```

Finally we can call the service.

```bash
gomu call helloworld Helloworld.Call '{"name": "John"}'
```

That's all you need to know to get started. Refer to the [Go Micro][1]
documentation for more info on developing services.

## Dependencies

You will need protoc-gen-micro for code generation

* [protobuf][4]
* [protoc-gen-go][5]
* [protoc-gen-micro][6]

```bash
# Download latest proto release
# https://github.com/protocolbuffers/protobuf/releases
go get -u google.golang.org/protobuf/proto
go install github.com/golang/protobuf/protoc-gen-go@latest
go install github.com/asim/go-micro/cmd/protoc-gen-micro/v3@latest
```

## Creating A Service

To create a new service, use the `gomu new service` command.

```bash
$ gomu new service helloworld
creating service helloworld

download protoc zip packages (protoc-$VERSION-$PLATFORM.zip) and install:

visit https://github.com/protocolbuffers/protobuf/releases/latest

download protobuf for go-micro:

go get -u google.golang.org/protobuf/proto
go install github.com/golang/protobuf/protoc-gen-go@latest
go install github.com/asim/go-micro/cmd/protoc-gen-micro/v3@latest

compile the proto file helloworld.proto:

cd helloworld
make proto tidy
```

To create a new function, use the `gomu new function` command. Functions differ
from services in that they exit after returning.

```bash
$ gomu new function helloworld
creating function helloworld

download protoc zip packages (protoc-$VERSION-$PLATFORM.zip) and install:

visit https://github.com/protocolbuffers/protobuf/releases/latest

download protobuf for go-micro:

go get -u google.golang.org/protobuf/proto
go install github.com/golang/protobuf/protoc-gen-go@latest
go install github.com/asim/go-micro/cmd/protoc-gen-micro/v3@latest

compile the proto file helloworld.proto:

cd helloworld
make proto tidy
```

### Jaeger

To create a new service with [Jaeger][7] integration, pass the `--jaeger` flag
to the `gomu new service` or `gomu new function` commands. You may configure
the Jaeger client using [environment variables][8].

```bash
gomu new service --jaeger helloworld
```

You may invoke `trace.NewSpan(context.Context).Finish()` to nest spans. For
example, consider the following handler implementing a greeter.

`handler/helloworld.go`

```go
package helloworld

import (
    "context"

    log "github.com/asim/go-micro/v3/logger"

    "helloworld/greeter"
    pb "helloworld/proto"
)

type Helloworld struct{}

func (e *Helloworld) Call(ctx context.Context, req pb.CallRequest, rsp *pb.CallResponse) error {
    log.Infof("Received Helloworld.Call request: %v", req)
    rsp.Msg = greeter.Greet(ctx, req.Name)
    return nil
}
```

`greeter/greeter.go`

```go
package greeter

import (
    "context"
    "fmt"

    "github.com/asim/go-micro/cmd/gomu/debug/trace"
)

func Greet(ctx context.Context, name string) string {
    defer trace.NewSpan(ctx).Finish()
    return fmt.Sprint("Hello " + name)
}
```

### Skaffold

To create a new service with [Skaffold][9] files, pass the `--skaffold` flag to
the `gomu new service` or `gomu new function` commands.

```bash
gomu new service --skaffold helloworld
```

## Running A Service

To run a service, use the `gomu run` command to build and run your service
continuously.

```bash
$ gomu run
2021-08-20 14:05:54  file=v3@v3.5.2/service.go:199 level=info Starting [service] helloworld
2021-08-20 14:05:54  file=server/rpc_server.go:820 level=info Transport [http] Listening on [::]:34531
2021-08-20 14:05:54  file=server/rpc_server.go:840 level=info Broker [http] Connected to 127.0.0.1:44975
2021-08-20 14:05:54  file=server/rpc_server.go:654 level=info Registry [mdns] Registering node: helloworld-45f43a6f-5fc0-4b0d-af73-e4a10c36ef54
```

### With Docker

To run a service with Docker, build the Docker image and run the Docker
container.

```bash
$ make docker
$ docker run helloworld:latest
2021-08-20 12:07:31  file=v3@v3.5.2/service.go:199 level=info Starting [service] helloworld
2021-08-20 12:07:31  file=server/rpc_server.go:820 level=info Transport [http] Listening on [::]:36037
2021-08-20 12:07:31  file=server/rpc_server.go:840 level=info Broker [http] Connected to 127.0.0.1:46157
2021-08-20 12:07:31  file=server/rpc_server.go:654 level=info Registry [mdns] Registering node: helloworld-31f58714-72f5-4d12-b2eb-98f66aea7a34
```

### With Skaffold

When you've created your service using the `--skaffold` flag, you may run the
Skaffold pipeline using the `skaffold` command.

```bash
skaffold dev
```

## Creating A Client

To create a new client, use the `gomu new client` command. The name is the
service you'd like to create a client project for.

```bash
$ gomu new client helloworld
creating client helloworld
cd helloworld-client
make tidy
```

You may optionally pass the fully qualified package name of the service you'd
like to create a client project for.

```bash
$ gomu new client github.com/auditemarlow/helloworld
creating client helloworld
cd helloworld-client
make tidy
```

## Running A Client

To run a client, use the `gomu run` command to build and run your client
continuously.

```bash
$ gomu run
2021-09-03 12:52:23  file=helloworld-client/main.go:33 level=info msg:"Hello John"
```

## Generating Files

To generate Go Micro project template files after the fact, use the `gomu
generate` command. It will place the generated files in the current working
directory.

```bash
$ gomu generate skaffold
skaffold project template files generated
```

## Listing Services

To list services, use the `gomu services` command.

```bash
$ gomu services
helloworld
```

## Describing A Service

To describe a service, use the `gomu describe service` command.

```bash
$ gomu describe service helloworld
{
  "name": "helloworld",
  "version": "latest",
  "metadata": null,
  "endpoints": [
    {
      "name": "Helloworld.Call",
      "request": {
        "name": "CallRequest",
        "type": "CallRequest",
        "values": [
          {
            "name": "name",
            "type": "string",
            "values": null
          }
        ]
      },
      "response": {
        "name": "CallResponse",
        "type": "CallResponse",
        "values": [
          {
            "name": "msg",
            "type": "string",
            "values": null
          }
        ]
      }
    }
  ],
  "nodes": [
    {
      "id": "helloworld-9660f06a-d608-43d9-9f44-e264ff63c554",
      "address": "172.26.165.161:45059",
      "metadata": {
        "broker": "http",
        "protocol": "mucp",
        "registry": "mdns",
        "server": "mucp",
        "transport": "http"
      }
    }
  ]
}
```

You may pass the `--format=yaml` flag to output a YAML formatted object.

```bash
$ gomu describe service --format=yaml helloworld
name: helloworld
version: latest
metadata: {}
endpoints:
- name: Helloworld.Call
  request:
    name: CallRequest
    type: CallRequest
    values:
    - name: name
      type: string
      values: []
  response:
    name: CallResponse
    type: CallResponse
    values:
    - name: msg
      type: string
      values: []
nodes:
- id: helloworld-9660f06a-d608-43d9-9f44-e264ff63c554
  address: 172.26.165.161:45059
  metadata:
    broker: http
    protocol: mucp
    registry: mdns
    server: mucp
    transport: http
```

## Calling A Service

To call a service, use the `gomu call` command. This will send a single request
and expect a single response.

```bash
$ gomu call helloworld Helloworld.Call '{"name": "John"}'
{"msg":"Hello John"}
```

To call a service's server stream, use the `gomu stream server` command. This
will send a single request and expect a stream of responses.

```bash
$ gomu stream server helloworld Helloworld.ServerStream '{"count": 10}'
{"count":0}
{"count":1}
{"count":2}
{"count":3}
{"count":4}
{"count":5}
{"count":6}
{"count":7}
{"count":8}
{"count":9}
```

To call a service's bidirectional stream, use the `gomu stream bidi` command.
This will send a stream of requests and expect a stream of responses.

```bash
$ gomu stream bidi helloworld Helloworld.BidiStream '{"stroke": 1}' '{"stroke": 2}' '{"stroke": 3}'
{"stroke":1}
{"stroke":2}
{"stroke":3}
```

## License

This software is published under the [MIT license][10].

[1]: https://github.com/asim/go-micro
[2]: https://golang.org/dl/
[3]: https://golang.org/cmd/go/#hdr-Compile_and_install_packages_and_dependencies
[4]: https://grpc.io/docs/protoc-installation/
[5]: https://micro.mu/github.com/golang/protobuf/protoc-gen-go
[6]: https://github.com/asim/go-micro/tree/master/cmd/protoc-gen-micro
[7]: https://www.jaegertracing.io/
[8]: https://github.com/jaegertracing/jaeger-client-go#environment-variables
[9]: https://skaffold.dev/
[10]: LICENSE
