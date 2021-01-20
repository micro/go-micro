module github.com/micro/go-micro/plugins/server/grpc/v2

go 1.15

require (
	github.com/golang/protobuf v1.4.3
	github.com/micro/go-micro/v2 v2.9.2-0.20201226154210-35d72660c801
	golang.org/x/net v0.0.0-20201224014010-6772e930b67b
	google.golang.org/genproto v0.0.0-20201214200347-8c77b98c765d
	google.golang.org/grpc v1.27.0
)

replace google.golang.org/grpc => google.golang.org/grpc v1.26.0
