module github.com/asim/go-micro/plugins/registry/etcd/v3

go 1.15

require (
	github.com/asim/go-micro/v3 v3.0.0-20210120135431-d94936f6c97c
	github.com/coreos/go-semver v0.3.0 // indirect
	github.com/coreos/pkg v0.0.0-20180928190104-399ea9e2e55f // indirect
	github.com/mitchellh/hashstructure v1.1.0
	go.etcd.io/etcd v3.3.25+incompatible // indirect
	go.uber.org/zap v1.16.0
	go.etcd.io/etcd/client/v3 v3.0.0-20210204162551-dae29bb719dd
)

replace (
	go.etcd.io/etcd/api/v3 v3.5.0-pre => go.etcd.io/etcd/api/v3 v3.0.0-20210204162551-dae29bb719dd
	go.etcd.io/etcd/pkg/v3 v3.5.0-pre => go.etcd.io/etcd/pkg/v3 v3.0.0-20210204162551-dae29bb719dd
)