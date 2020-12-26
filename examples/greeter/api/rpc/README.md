# RPC API

This is an example of using an RPC based API

## Getting Started

### Run Micro API

```shell
micro api
```

### Run Greeter Service

```shell
go run greeter/srv/main.go
```

### Run Greeter API

```shell
go run rpc.go
```

### Curl API

```shell
curl -H 'Content-Type: application/json' -d '{"name": "Asim"}' http://localhost:8080/greeter/hello
```

Output

```
{
  "msg": "Hello Asim"
}
```
