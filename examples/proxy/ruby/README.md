# Ruby

- proxy.rb: methods to call proxy
- rpc_{client,server}.rb: RPC client/server
- http_{client,server}.rb: HTTP client/server

## RPC Example

Run proxy
```shell
micro proxy
```

Run server
```shell
# serves Say.Hello
ruby rpc_server.rb
```

Run client
```shell
# calls go.micro.srv.greeter Say.Hello
ruby rpc_client.rb
```

## HTTP Example

Run proxy with proxy handler
```shell
micro proxy --handler=http
```

Run server
```shell
# serves /greeter
ruby http_server.rb
```

Run client
```shell
# calls /greeter
ruby http_client.rb
```
