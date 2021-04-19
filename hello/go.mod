module hello

go 1.13

// This can be removed once etcd becomes go gettable, version 3.4 and 3.5 is not,
// see https://github.com/etcd-io/etcd/issues/11154 and https://github.com/etcd-io/etcd/issues/11931.
replace (
	github.com/micro/go-micro/v2 => ../
	google.golang.org/grpc => google.golang.org/grpc v1.26.0
)

require (
	github.com/golang/protobuf v1.5.2
	github.com/micro/go-micro/v2 v2.0.0-00010101000000-000000000000
	google.golang.org/protobuf v1.26.0
)
