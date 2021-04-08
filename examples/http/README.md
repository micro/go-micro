# http server and http client demo

An example http application

## Contents

- **srv** - a http server as server of go-mirco service
- **cli** - a http client that call http server
- **rpcli** - a http client that call rpc server


## Run Service
Start http server
```shell
go run srv/main.go
```

## Client

Call http client
```shell
go run cli/main.go

```


## Run rpc Service
Start greeter service
```shell
go run ../greeter/srv/main.go
```

## Client
http client call rpc service
```shell
go run rpccli/main.go
```
