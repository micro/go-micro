package main

import (
	_ "github.com/asim/go-micro/plugins/broker/kafka/v4"
	_ "github.com/asim/go-micro/plugins/broker/mqtt/v4"
	_ "github.com/asim/go-micro/plugins/broker/nats/v4"
	_ "github.com/asim/go-micro/plugins/broker/rabbitmq/v4"
	_ "github.com/asim/go-micro/plugins/broker/redis/v4"

	_ "github.com/asim/go-micro/plugins/registry/consul/v4"
	_ "github.com/asim/go-micro/plugins/registry/etcd/v4"
	_ "github.com/asim/go-micro/plugins/registry/eureka/v4"
	_ "github.com/asim/go-micro/plugins/registry/gossip/v4"
	_ "github.com/asim/go-micro/plugins/registry/kubernetes/v4"
	_ "github.com/asim/go-micro/plugins/registry/nacos/v4"
	_ "github.com/asim/go-micro/plugins/registry/nats/v4"
	_ "github.com/asim/go-micro/plugins/registry/zookeeper/v4"
)
