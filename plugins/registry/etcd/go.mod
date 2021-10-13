module github.com/asim/go-micro/plugins/registry/etcd/v4

go 1.16

require (
	github.com/mitchellh/hashstructure v1.1.0
	go-micro.dev/v4 v4.1.0
	go.etcd.io/etcd/api/v3 v3.5.0
	go.etcd.io/etcd/client/v3 v3.5.0
	go.uber.org/zap v1.17.0
)

replace go-micro.dev/v4 => ../../../../go-micro
