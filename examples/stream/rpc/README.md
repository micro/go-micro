# Stream

This is an example of a streaming service and two clients, a streaming rpc client and a client using websockets.

## Contents

- server - is the service
- client - is the rpc client
- web - is the websocket client

## Run the example

Run the service

```shell
go run server/main.go
```

Run the client

```shell
go run client/main.go
```

Run the micro web reverse proxy for the websocket client

``` shell
micro web
```

Run the websocket client

```shell
cd web # must be in the web directory to serve static files.
go run main.go
```

Visit http://localhost:8082/stream and send a request!

And that's all there is to it.
