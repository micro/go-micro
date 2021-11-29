package main

import (
	_ "github.com/asim/go-micro/plugins/registry/consul/v4"
	_ "github.com/asim/go-micro/plugins/registry/etcd/v4"
	_ "github.com/asim/go-micro/plugins/registry/eureka/v4"
	_ "github.com/asim/go-micro/plugins/registry/gossip/v4"
	_ "github.com/asim/go-micro/plugins/registry/kubernetes/v4"
	_ "github.com/asim/go-micro/plugins/registry/nacos/v4"
	_ "github.com/asim/go-micro/plugins/registry/nats/v4"
	_ "github.com/asim/go-micro/plugins/registry/zookeeper/v4"
)
