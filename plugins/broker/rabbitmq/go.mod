module github.com/asim/go-micro/plugins/broker/rabbitmq/v4

go 1.16

require (
	github.com/google/uuid v1.2.0
	github.com/streadway/amqp v1.0.0
	go-micro.dev/v4 v4.1.0
)

replace go-micro.dev/v4 => ../../../../go-micro
