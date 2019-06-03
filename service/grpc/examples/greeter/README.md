# Greeter Service

An example Go-Micro based gRPC service

## What's here?

- **server** - a gRPC greeter service
- **client** - a gRPC client that calls the service once
- **function** - a gRPC greeter function，more about [Function](https://micro.mu/docs/writing-a-go-function.html)
- **gateway** - a grpc-gateway

## Test Service

Run Service
```
$ go run server/main.go --registry=mdns
2016/11/03 18:41:22 Listening on [::]:55194
2016/11/03 18:41:22 Broker Listening on [::]:55195
2016/11/03 18:41:22 Registering node: go.micro.srv.greeter-1e200612-a1f5-11e6-8e84-68a86d0d36b6
```

Test Service
```
$ go run client/main.go --registry=mdns
Hello John
```

## Test Function

Run function

```
go run function/main.go --registry=mdns
```

Query function

```
go run client/main.go --registry=mdns --service_name="go.micro.fnc.greeter"
```

## Test Gateway

Run server with address set

```
go run server/main.go --registry=mdns --server_address=localhost:9090
```

Run gateway

```
go run gateway/main.go
```

Curl gateway

```
curl -d '{"name": "john"}' http://localhost:8080/greeter/hello
```

## i18n

### [中文](README_cn.md)