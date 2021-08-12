module github.com/asim/go-micro/plugins/broker/googlepubsub/v3

go 1.16

require (
	cloud.google.com/go/pubsub v1.12.0
	github.com/asim/go-micro/v3 v3.5.2-0.20210630062103-c13bb07171bc
	github.com/google/uuid v1.2.0
	google.golang.org/api v0.49.0
	google.golang.org/grpc v1.38.0
)

replace github.com/asim/go-micro/v3 => ../../../../go-micro
