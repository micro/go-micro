module github.com/asim/go-micro/plugins/broker/gocloud/v3

go 1.16

require (
	github.com/asim/go-micro/v3 v3.5.2-0.20210630062103-c13bb07171bc
	github.com/streadway/amqp v1.0.0
	gocloud.dev v0.17.0
	gocloud.dev/pubsub/rabbitpubsub v0.17.0
)

replace github.com/asim/go-micro/v3 => ../../../../go-micro
