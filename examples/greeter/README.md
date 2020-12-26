# Greeter

An example Greeter application

## Contents

- **srv** - an RPC greeter service
- **cli** - an RPC client that calls the service once
- **api** - examples of RPC API and RESTful API
- **web** - how to use go-web to write web services

## Run Service

Start go.micro.srv.greeter
```shell
go run srv/main.go
```

## Client

Call go.micro.srv.greeter via client
```shell
go run cli/main.go
```

Examples of client usage via other languages can be found in the client directory.

## API

HTTP based requests can be made via the micro API. Micro logically separates API services from backend services. By default the micro API 
accepts HTTP requests and converts to *api.Request and *api.Response types. Find them here [micro/api/proto](https://github.com/micro/micro/tree/master/api/proto).

Run the go.micro.api.greeter API Service
```shell
go run api/api.go 
```

Run the micro API
```shell
micro api --handler=api
```

Call go.micro.api.greeter via API
```shell
curl http://localhost:8080/greeter/say/hello?name=John
```

Examples of other API handlers can be found in the API directory.

## Web

The micro web is a web dashboard and reverse proxy to run web apps as microservices.

Run go.micro.web.greeter
```
go run web/web.go 
```

Run the micro web
```shell
micro web
```

Browse to http://localhost:8082/greeter
