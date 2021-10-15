module github.com/asim/go-micro/examples/v4

go 1.16

require (
	github.com/asim/go-micro/plugins/client/http/v4 v4.0.0-20211013123517-8cad88edae00
	github.com/asim/go-micro/plugins/config/encoder/toml/v4 v4.0.0-20211013123517-8cad88edae00
	github.com/asim/go-micro/plugins/config/encoder/yaml/v4 v4.0.0-20211013123517-8cad88edae00
	github.com/asim/go-micro/plugins/config/source/grpc/v4 v4.0.0-20211013123517-8cad88edae00
	github.com/asim/go-micro/plugins/server/http/v4 v4.0.0-20211013123517-8cad88edae00
	github.com/asim/go-micro/plugins/wrapper/select/roundrobin/v4 v4.0.0-20211013123517-8cad88edae00
	github.com/asim/go-micro/plugins/wrapper/select/shard/v4 v4.0.0-20211013123517-8cad88edae00
	github.com/gin-gonic/gin v1.7.0
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/golang/protobuf v1.5.2
	github.com/grpc-ecosystem/grpc-gateway v1.16.0
	github.com/pborman/uuid v1.2.1
	github.com/urfave/cli/v2 v2.3.0
	go-micro.dev/v4 v4.1.1-0.20211013123517-8cad88edae00
	golang.org/x/net v0.0.0-20210614182718-04defd469f4e
	google.golang.org/genproto v0.0.0-20210624195500-8bfb893ecb84
	google.golang.org/grpc v1.38.0
	google.golang.org/protobuf v1.26.0
)
