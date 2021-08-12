module github.com/asim/go-micro/plugins/transport/nats/v3

go 1.16

require (
	github.com/asim/go-micro/v3 v3.5.2-0.20210630062103-c13bb07171bc
	github.com/go-log/log v0.2.0
	github.com/nats-io/nats-server/v2 v2.3.0 // indirect
	github.com/nats-io/nats.go v1.11.0
)

replace github.com/asim/go-micro/v3 => ../../../../go-micro
