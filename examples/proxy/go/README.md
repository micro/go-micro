# Go

This Go example uses vanilla net/http and the proxy

- proxy.go: methods to call proxy
- rpc_{client,server}.go: RPC client/server
- http_{client,server}.go: HTTP client/server

## RPC Example

Run proxy
```shell
micro proxy
```

Run server
```shell
# serves Say.Hello
go run rpc_server.go proxy.go
```

Run client
```shell
# calls go.micro.srv.greeter Say.Hello
go run rpc_client.go proxy.go
```

## HTTP Example

Run proxy with proxy handler
```shell
micro proxy --handler=http
```

Run server
```shell
# serves /greeter
go run http_server.go proxy.go
```

Run client
```shell
# calls /greeter
go run http_client.go proxy.go
```
