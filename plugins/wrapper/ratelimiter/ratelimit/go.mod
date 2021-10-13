module github.com/asim/go-micro/plugins/wrapper/ratelimiter/ratelimit/v4

go 1.16

require (
	github.com/asim/go-micro/plugins/broker/memory/v4 v4.0.0-20211013123123-62801c3d6883
	github.com/asim/go-micro/plugins/registry/memory/v4 v4.0.0-20211013123123-62801c3d6883
	github.com/asim/go-micro/plugins/transport/memory/v4 v4.0.0-20211013123123-62801c3d6883
	github.com/juju/ratelimit v1.0.1
	go-micro.dev/v4 v4.1.0
)

replace (
	github.com/asim/go-micro/plugins/broker/memory/v4 => ../../../../plugins/broker/memory
	github.com/asim/go-micro/plugins/registry/memory/v4 => ../../../../plugins/registry/memory
	github.com/asim/go-micro/plugins/transport/memory/v4 => ../../../../plugins/transport/memory
	go-micro.dev/v4 => ../../../../../go-micro
)
