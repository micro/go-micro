module github.com/asim/go-micro/plugins/registry/etcd/v3

go 1.13

require (
	github.com/asim/go-micro/v3 v3.5.0
	github.com/mitchellh/hashstructure v1.1.0
	go.etcd.io/etcd/api/v3 v3.5.0-pre
	go.etcd.io/etcd/client/v3 v3.0.0-20210204162551-dae29bb719dd
	go.uber.org/zap v1.16.0
)

replace (
	go.etcd.io/etcd/api/v3 => go.etcd.io/etcd/api/v3 v3.0.0-20210204162551-dae29bb719dd
	go.etcd.io/etcd/pkg/v3 => go.etcd.io/etcd/pkg/v3 v3.0.0-20210204162551-dae29bb719dd
	google.golang.org/grpc => google.golang.org/grpc v1.29.1
)
