module github.com/asim/go-micro/plugins/wrapper/breaker/hystrix/v3

go 1.16

require (
	github.com/afex/hystrix-go v0.0.0-20180502004556-fa1af6a1f4f5
	github.com/asim/go-micro/plugins/registry/memory/v3 v3.5.1
	github.com/asim/go-micro/v3 v3.5.1
)

replace (
	github.com/asim/go-micro/plugins/registry/memory/v3 => ../../../../plugins/registry/memory
	github.com/asim/go-micro/v3 => ../../../../../go-micro
)
