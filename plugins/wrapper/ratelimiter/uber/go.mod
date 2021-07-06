module github.com/asim/go-micro/plugins/wrapper/ratelimiter/uber/v3

go 1.16

require (
	github.com/asim/go-micro/v3 v3.5.2-0.20210630062103-c13bb07171bc
	go.uber.org/ratelimit v0.2.0
)

replace github.com/asim/go-micro/v3 => ../../../../../go-micro
