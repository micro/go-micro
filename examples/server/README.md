# Service

An example Go service running with go-micro

## Contents

- main.go - initialises and runs the the server
- handler - is an example RPC request handler for the Server
- proto - contains the protobuf defintion for the Server API
- subscriber - is a handler for subscribing via the Server
- wrapper - demonstrates use of a server HandlerWrapper
- codegen - shows how to use codegenerated registration to reduce boilerplate

## Usage

Run Service
```
$ go run server/main.go
I0525 18:06:14.471489   83304 server.go:117] Starting server go.micro.srv.example id go.micro.srv.example-59b6e0ab-0300-11e5-b696-68a86d0d36b6
I0525 18:06:14.474960   83304 rpc_server.go:126] Listening on [::]:62216
I0525 18:06:14.474997   83304 server.go:99] Registering node: go.micro.srv.example-59b6e0ab-0300-11e5-b696-68a86d0d36b6
```

Test Service
```
$ go run client/main.go 
go.micro.srv.example-59b6e0ab-0300-11e5-b696-68a86d0d36b6: Hello John
```
