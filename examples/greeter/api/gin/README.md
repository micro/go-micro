# Gin API Example

This is an example of how to serve an API using gin

## Getting Started

### Run the Micro API

```
$ micro api --handler=http
```

### Run the Greeter Service

```
$ go run greeter/srv/main.go
```

### Run the Greeter API

```
$ go run gin.go
Listening on [::]:64738
```

### Curl the API

Test the index
```
curl http://localhost:8080/greeter
{
  "message": "Hi, this is the Greeter API"
}
```

Test a resource
```
 curl http://localhost:8080/greeter/asim
{
  "msg": "Hello asim"
}
```
