module github.com/asim/go-micro/plugins/broker/stan/v3

go 1.16

require (
	github.com/asim/go-micro/v3 v3.5.2-0.20210629124054-4929a7c16ecc
	github.com/google/uuid v1.2.0
	github.com/nats-io/nats-server/v2 v2.3.0 // indirect
	github.com/nats-io/nats-streaming-server v0.22.0 // indirect
	github.com/nats-io/stan.go v0.9.0
)

replace github.com/asim/go-micro/v3 => ../../../../go-micro
