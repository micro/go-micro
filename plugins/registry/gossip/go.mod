module github.com/asim/go-micro/plugins/registry/gossip/v3

go 1.16

require (
	github.com/asim/go-micro/v3 v3.5.2-0.20210630062103-c13bb07171bc
	github.com/golang/protobuf v1.5.2
	github.com/google/uuid v1.2.0
	github.com/hashicorp/memberlist v0.1.5
	github.com/mitchellh/hashstructure v1.1.0
)

replace github.com/asim/go-micro/v3 => ../../../../go-micro
