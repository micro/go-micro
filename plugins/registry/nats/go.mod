module github.com/asim/go-micro/plugins/registry/nats/v4

go 1.16

require (
	github.com/nats-io/nats-server/v2 v2.1.9 // indirect
	github.com/nats-io/nats.go v1.10.0
	go-micro.dev/v4 v4.2.1
)

replace go-micro.dev/v4 => ../../../../go-micro
