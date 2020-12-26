# Beego API Example

This is an example of how to serve an API using beego

## Getting Started

### Run the Micro API

```
$ micro api --handler=http
```

### Run the Greeter Service

```shell
$ go run greeter/srv/main.go
```

### Run the Greeter API

```shell
$ go run beego.go
```

### Curl API

Test the index

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
