# Template Service

An example Go service running with go-micro

### Prerequisites

Install Consul
[https://www.consul.io/intro/getting-started/install.html](https://www.consul.io/intro/getting-started/install.html)

Run Consul
```
$ consul agent -server -bootstrap-expect 1 -data-dir /tmp/consul
```

Run Service
```
$ go run main.go

I0523 12:21:09.506998   77096 server.go:113] Starting server go.micro.service.template id go.micro.service.template-cfc481fc-013d-11e5-bcdc-68a86d0d36b6
I0523 12:21:09.507281   77096 rpc_server.go:112] Listening on [::]:51868
I0523 12:21:09.507329   77096 server.go:95] Registering node: go.micro.service.template-cfc481fc-013d-11e5-bcdc-68a86d0d36b6
```

Test Service
```
$ go run go-micro/examples/service_client.go

go.micro.service.template-cfc481fc-013d-11e5-bcdc-68a86d0d36b6: Hello John
```
