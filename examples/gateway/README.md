# GRPC Gateway

This directory contains a grpc gateway generated using [grpc-ecosystem/grpc-gateway](https://github.com/grpc-ecosystem/grpc-gateway).

Services written with [asim/go-micro/service/grpc](https://github.com/asim/go-micro/service/grpc) are fully compatible with the grpc-gateway and any other 
grpc services.

Go to [grpc-ecosystem/grpc-gateway](https://github.com/grpc-ecosystem/grpc-gateway) for details on how to generate gateways. We 
have generated the gateway from the same proto as the greeter server but with additional options for the gateway.

## Usage

Run the go.micro.srv.greeter service

```
go run ../greeter/srv/main.go --server_address=localhost:9090
```

Run the gateway

```
go run main.go
```

Curl your request at the gateway (localhost:8080)

```
curl -d '{"name": "john"}' http://localhost:8080/greeter/hello
```
