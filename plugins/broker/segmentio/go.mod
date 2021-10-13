module github.com/asim/go-micro/plugins/broker/segmentio/v4

go 1.16

require (
	github.com/asim/go-micro/plugins/broker/kafka/v4 master
	github.com/asim/go-micro/plugins/codec/segmentio/v4 master
	go-micro.dev/v4 v4.1.0
	github.com/google/uuid v1.2.0
	github.com/segmentio/kafka-go v0.4.16
)

replace (
	github.com/asim/go-micro/plugins/broker/kafka/v4 => ../kafka
	github.com/asim/go-micro/plugins/codec/segmentio/v4 => ../../codec/segmentio
	go-micro.dev/v4 => ../../../../go-micro
)
