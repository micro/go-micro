module github.com/asim/go-micro/plugins/wrapper/breaker/gobreaker/v3

go 1.16

require (
	github.com/asim/go-micro/plugins/registry/memory/v3 v3.0.0-20210630062103-c13bb07171bc
	go-micro.dev/v4 v4.0.0
	github.com/sony/gobreaker v0.4.1
)

replace (
	github.com/asim/go-micro/plugins/registry/memory/v3 => ../../../../plugins/registry/memory
	github.com/asim/go-micro/v3 => ../../../../../go-micro
)
