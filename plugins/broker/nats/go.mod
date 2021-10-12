module github.com/asim/go-micro/plugins/broker/nats/v3

go 1.16

require (
	go-micro.dev/v4 v4.1.0
	github.com/nats-io/nats-server/v2 v2.3.1 // indirect
	github.com/nats-io/nats.go v1.11.1-0.20210623165838-4b75fc59ae30
)

replace github.com/asim/go-micro/v3 => ../../../../go-micro
