# Service

This is an example of creating a micro service using the top level interface.

## Prereqs

Micro services need a discovery system so they can find each other. Micro uses consul by default but 
its easily swapped out with etcd, kubernetes, or various other systems. We'll run consul for convenience.

1. Follow the install instructions - [https://www.consul.io/intro/getting-started/install.html](https://www.consul.io/intro/getting-started/install.html)

2. Run Consul

```shell
$ consul agent -server -bootstrap-expect 1 -data-dir /tmp/consul
```

## Run the example

1. Get the service

```shell
go get github.com/micro/go-micro/examples/service
```

2. Run the server

```shell
$GOPATH/bin/service
```

3. Run the client

```shell
$GOPATH/bin/service --client
```

And that's all there is to it.
