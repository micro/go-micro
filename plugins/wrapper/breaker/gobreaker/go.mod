module github.com/asim/go-micro/plugins/wrapper/breaker/gobreaker/v4

go 1.16

require (
	github.com/sony/gobreaker v0.4.1
	go-micro.dev/v4 v4.2.1
)

replace go-micro.dev/v4 => ../../../../../go-micro
