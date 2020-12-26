# REST API

This is an example of how to serve REST behind the API using go-restful

## Getting Started

### Run Micro API

```
$ micro api --handler=http
```

### Run Greeter Service

```shell
go run greeter/srv/main.go
```

### Run Greeter API

```shell
go run rest.go
```

### Curl API

```shell
curl http://localhost:8080/greeter
```

Output

```json
{
  "message": "Hi, this is the Greeter API"
}
```

Test a resource

```shell
 curl http://localhost:8080/greeter/asim
```

Output
```json
{
  "msg": "Hello asim"
}
```
