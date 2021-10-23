module github.com/asim/go-micro/plugins/registry/gossip/v4

go 1.16

require (
	github.com/golang/protobuf v1.5.2
	github.com/google/uuid v1.2.0
	github.com/hashicorp/memberlist v0.1.5
	github.com/mitchellh/hashstructure v1.1.0
	go-micro.dev/v4 v4.2.1
)

replace go-micro.dev/v4 => ../../../../go-micro
