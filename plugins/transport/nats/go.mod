module github.com/asim/go-micro/plugins/transport/nats/v4

go 1.16

require (
	github.com/go-log/log v0.2.0
	github.com/nats-io/nats-server/v2 v2.3.0 // indirect
	github.com/nats-io/nats.go v1.11.0
	go-micro.dev/v4 v4.1.0
)

replace go-micro.dev/v4 => ../../../../go-micro
