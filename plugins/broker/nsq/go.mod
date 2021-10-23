module github.com/asim/go-micro/plugins/broker/nsq/v4

go 1.16

require (
	github.com/google/uuid v1.2.0
	github.com/nsqio/go-nsq v1.0.8
	go-micro.dev/v4 v4.2.1
)

replace go-micro.dev/v4 => ../../../../go-micro
