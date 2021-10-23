module github.com/asim/go-micro/plugins/sync/consul/v4

go 1.16

require (
	github.com/hashicorp/consul/api v1.9.0
	github.com/hashicorp/go-hclog v0.16.2
	go-micro.dev/v4 v4.2.1
)

replace go-micro.dev/v4 => ../../../../go-micro
