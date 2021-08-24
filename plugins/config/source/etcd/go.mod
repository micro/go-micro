module github.com/asim/go-micro/plugins/config/source/etcd/v3

go 1.16

require (
	github.com/asim/go-micro/v3 v3.5.2-0.20210630062103-c13bb07171bc
	go.etcd.io/etcd/api/v3 v3.5.0
	go.etcd.io/etcd/client/v3 v3.5.0
)

replace github.com/asim/go-micro/v3 => ../../../../../go-micro
