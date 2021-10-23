module github.com/asim/go-micro/examples/v4

go 1.16

replace (
	github.com/asim/go-micro/plugins/client/grpc/v4 => ../plugins/client/grpc
	github.com/asim/go-micro/plugins/transport/grpc/v4 => ../plugins/transport/grpc
	go-micro.dev/v4 => ../../go-micro
)

require (
	github.com/asim/go-micro/plugins/client/http/v4 v4.0.0-20211022143028-f96b48dad9f9
	github.com/asim/go-micro/plugins/config/encoder/toml/v4 v4.0.0-20211022143028-f96b48dad9f9
	github.com/asim/go-micro/plugins/config/encoder/yaml/v4 v4.0.0-20211022143028-f96b48dad9f9
	github.com/asim/go-micro/plugins/config/source/grpc/v4 v4.0.0-20211022143028-f96b48dad9f9
	github.com/asim/go-micro/plugins/server/http/v4 v4.0.0-20211022143028-f96b48dad9f9
	github.com/asim/go-micro/plugins/wrapper/select/roundrobin/v4 v4.0.0-20211022143028-f96b48dad9f9
	github.com/asim/go-micro/plugins/wrapper/select/shard/v4 v4.0.0-20211022143028-f96b48dad9f9
	github.com/gin-gonic/gin v1.7.4
	github.com/golang/glog v1.0.0
	github.com/golang/protobuf v1.5.2
	github.com/grpc-ecosystem/grpc-gateway v1.16.0
	github.com/pborman/uuid v1.2.1
	github.com/urfave/cli/v2 v2.3.0
	go-micro.dev/v4 v4.1.0
	golang.org/x/net v0.0.0-20211020060615-d418f374d309
	google.golang.org/genproto v0.0.0-20211021150943-2b146023228c
	google.golang.org/grpc v1.41.0
	google.golang.org/protobuf v1.27.1
)
