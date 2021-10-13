module github.com/asim/go-micro/plugins/wrapper/breaker/gobreaker/v4

go 1.16

require (
	github.com/asim/go-micro/plugins/registry/memory/v4 master
	go-micro.dev/v4 v4.1.0
	github.com/sony/gobreaker v0.4.1
)

replace (
	github.com/asim/go-micro/plugins/registry/memory/v4 => ../../../../plugins/registry/memory
	go-micro.dev/v4 => ../../../../../go-micro
)
