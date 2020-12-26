# API

This example makes use of the "api" handler.

The api expects you use the [api.Request/Response](https://github.com/micro/go-api/blob/master/proto/api.proto) protos.

The micro api request handler gives you full control over the http request and response while still leveraging RPC and 
any transport plugins that use other protocols beyond http in your stack such as grpc, nats, kafka.

## Usage

Run the micro API

```
micro api --handler=api
```

Run this example

```
go run api.go
```


## Calling the service

Make a GET request to /example/call which will call go.micro.api.example Example.Call

```
curl "http://localhost:8080/example/call?name=john"
```

Make a POST request to /example/foo/bar which will call go.micro.api.example Foo.Bar

```
curl -H 'Content-Type: application/json' -d '{}' http://localhost:8080/example/foo/bar
```

## Set Namespace

Run the micro API with custom namespace

```
micro api --handler=api --namespace=com.foobar.api
```

or
```
MICRO_API_NAMESPACE=com.foobar.api micro api --handler=api
```

Set service name with the namespace

```
service := micro.NewService(
        micro.Name("com.foobar.api.example"),
)
```   
