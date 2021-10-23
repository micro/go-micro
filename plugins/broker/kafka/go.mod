module github.com/asim/go-micro/plugins/broker/kafka/v4

go 1.16

require (
	github.com/Shopify/sarama v1.29.1
	github.com/google/uuid v1.2.0
	go-micro.dev/v4 v4.2.1
)

replace go-micro.dev/v4 => ../../../../go-micro
