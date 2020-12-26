# Python

- proxy.py: methods to call proxy
- rpc_{client,server}.py: RPC client/server
- http_{client,server}.py: HTTP client/server

## RPC Example

Run proxy
```shell
micro proxy
```

Run server
```shell
# serves Say.Hello
python rpc_server.py
```

Run client
```shell
# calls go.micro.srv.greeter Say.Hello
python rpc_client.py
```

## HTTP Example

Run proxy with proxy handler
```shell
micro proxy --handler=http
```

Run server
```shell
# serves /greeter
python http_server.py
```

Run client
```shell
# calls /greeter
python http_client.py
```
