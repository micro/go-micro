module github.com/asim/go-micro/plugins/registry/consul/v3

go 1.16

require (
	go-micro.dev/v4 v4.0.0
	github.com/hashicorp/consul/api v1.9.0
	github.com/mitchellh/hashstructure v1.1.0
)

replace github.com/asim/go-micro/v3 => ../../../../go-micro
