module github.com/asim/go-micro/plugins/store/file/v4

go 1.16

require (
	github.com/davecgh/go-spew v1.1.1
	github.com/kr/pretty v0.2.1
	go-micro.dev/v4 v4.1.0
	go.etcd.io/bbolt v1.3.6
)

replace go-micro.dev/v4 => ../../../../go-micro
