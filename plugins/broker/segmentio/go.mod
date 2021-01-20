module github.com/asim/go-micro/plugins/broker/segmentio/v3

go 1.13

require (
	github.com/asim/go-micro/plugins/broker/kafka/v3 v3.0.0-00010101000000-000000000000
	github.com/asim/go-micro/plugins/codec/segmentio/v3 v3.0.0-00010101000000-000000000000
	github.com/asim/go-micro/v3 v3.0.0-20210120135431-d94936f6c97c
	github.com/google/uuid v1.1.1
	github.com/segmentio/kafka-go v0.3.7
)

replace github.com/asim/go-micro/plugins/broker/kafka/v3 => ../../broker/kafka

replace github.com/asim/go-micro/plugins/codec/segmentio/v3 => ../../codec/segmentio
