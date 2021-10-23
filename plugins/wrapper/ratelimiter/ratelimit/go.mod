module github.com/asim/go-micro/plugins/wrapper/ratelimiter/ratelimit/v4

go 1.16

require (
	github.com/juju/ratelimit v1.0.1
	go-micro.dev/v4 v4.2.1
)

replace go-micro.dev/v4 => ../../../../../go-micro
