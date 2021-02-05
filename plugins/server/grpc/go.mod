module github.com/asim/go-micro/plugins/server/grpc/v3

go 1.15

require (
	github.com/asim/go-micro/plugins/broker/memory/v3 v3.0.0-20210202145831-070250155285
	github.com/asim/go-micro/plugins/client/grpc/v3 v3.0.0-20210205090925-e8167a8b79ed
	github.com/asim/go-micro/plugins/registry/memory/v3 v3.0.0-20210202145831-070250155285
	github.com/asim/go-micro/plugins/transport/grpc/v3 v3.0.0-20210202145831-070250155285
	github.com/asim/go-micro/v3 v3.0.0-20210120135431-d94936f6c97c
	github.com/golang/protobuf v1.4.3
	golang.org/x/net v0.0.0-20201224014010-6772e930b67b
	google.golang.org/genproto v0.0.0-20201214200347-8c77b98c765d
	google.golang.org/grpc v1.27.0
)

replace google.golang.org/grpc => google.golang.org/grpc v1.26.0
