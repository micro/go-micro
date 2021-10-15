module github.com/asim/go-micro/plugins/registry/zookeeper/v4

go 1.16

require (
	github.com/go-zookeeper/zk v1.0.2
	github.com/mitchellh/hashstructure v1.1.0
	go-micro.dev/v4 v4.1.0
)

replace go-micro.dev/v4 => ../../../../go-micro
