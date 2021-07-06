module github.com/asim/go-micro/plugins/registry/nats/v3

go 1.16

require (
	github.com/asim/go-micro/v3 v3.5.2-0.20210630062103-c13bb07171bc
	github.com/nats-io/nats-server/v2 v2.1.9 // indirect
	github.com/nats-io/nats.go v1.10.0
)

replace github.com/asim/go-micro/v3 => ../../../../go-micro
