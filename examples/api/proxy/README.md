# Proxy API

This is an example of using the micro api as a http proxy.

Using the api as a http proxy gives you complete control over what languages or libraries to use 
at the API layer. In this case we're using go-web to easily register http services.

## Usage

Run micro api with http proxy handler

```
micro api --handler=http
```

Run this proxy service

```
go run proxy.go
```

Make a GET request to /example/call which will call go.micro.api.example Example.Call

```
curl "http://localhost:8080/example/call?name=john"
```

Make a POST request to /example/foo/bar which will call go.micro.api.example Foo.Bar

```
curl -H 'Content-Type: application/json' -d '{}' http://localhost:8080/example/foo/bar
```
