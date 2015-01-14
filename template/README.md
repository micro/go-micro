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

1416690099281057746 [Debug] Rpc handler /_rpc
1416690099281092588 [Debug] Starting server go.micro.service.template id go.micro.service.template-c0bfcb44-728a-11e4-b099-68a86d0d36b6
1416690099281192941 [Debug] Listening on [::]:58264
1416690099281215346 [Debug] Registering go.micro.service.template-c0bfcb44-728a-11e4-b099-68a86d0d36b6
```

Test Service
```
$ go run go-micro/examples/service_client.go

go.micro.service.template-c0bfcb44-728a-11e4-b099-68a86d0d36b6: Hello John
```
