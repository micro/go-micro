module github.com/asim/go-micro/plugins/config/source/consul/v4

go 1.16

require (
	github.com/hashicorp/consul/api v1.9.0
	go-micro.dev/v4 v4.1.0
)

replace go-micro.dev/v4 => ../../../../../go-micro
