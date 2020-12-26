# Meta API

This example makes use of the micro API metadata handler.

This will allow us to write standard go-micro services and set the handler/endpoint via service discovery metadata.

## Usage

Run the micro API

```
micro api
```

Run this example. Note endpoint metadata when registering the handler

```
go run meta.go
```

Make a POST request to /example which will call go.micro.api.example Example.Call

```
curl -H 'Content-Type: application/json' -d '{"name": "john"}' "http://localhost:8080/example"
```

Make a POST request to /foo/bar which will call go.micro.api.example Foo.Bar

```
curl -H 'Content-Type: application/json' -d '{}' http://localhost:8080/foo/bar
```
