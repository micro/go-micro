# API

This directory showcases API services which sit behind the micro api and serve a public facing API

## Services

- [**api**](api.go) - RPC api with [api.Request](https://github.com/micro/go-micro/blob/master/api/proto/api.proto#L11L18) and [api.Response](https://github.com/micro/go-micro/blob/master/api/proto/api.proto#L21L25) (Micro api handler should be set to --handler=api)
- [**beego**](beego) - using beego
- [**gin**](gin) - using gin server
- [**graphql**](graphql) - using graphql
- [**rest**](rest) - using go-restful
- [**rpc**](rpc) - using RPC
