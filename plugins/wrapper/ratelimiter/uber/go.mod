module github.com/asim/go-micro/plugins/wrapper/ratelimiter/uber/v4

go 1.16

require (
	go-micro.dev/v4 v4.2.1
	go.uber.org/ratelimit v0.2.0
)

replace go-micro.dev/v4 => ../../../../../go-micro
