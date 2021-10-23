module github.com/asim/go-micro/plugins/registry/consul/v4

go 1.16

require (
	github.com/hashicorp/consul/api v1.9.0
	github.com/mitchellh/hashstructure v1.1.0
	go-micro.dev/v4 v4.2.1
)

replace go-micro.dev/v4 => ../../../../go-micro
