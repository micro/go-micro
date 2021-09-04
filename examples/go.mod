module github.com/asim/go-micro/examples/v3

go 1.16

require (
	github.com/asim/go-micro/plugins/client/http/v3 v3.0.0-20210403073940-e7a7e3a05092
	github.com/asim/go-micro/plugins/config/encoder/toml/v3 v3.0.0-20210403073940-e7a7e3a05092
	github.com/asim/go-micro/plugins/config/encoder/yaml/v3 v3.0.0-20210804083901-3e0411a3f61b
	github.com/asim/go-micro/plugins/config/source/grpc/v3 v3.0.0-20210403073940-e7a7e3a05092
	github.com/asim/go-micro/plugins/server/http/v3 v3.0.0-20210403073940-e7a7e3a05092
	github.com/asim/go-micro/plugins/wrapper/select/roundrobin/v3 v3.0.0-20210403073940-e7a7e3a05092
	github.com/asim/go-micro/plugins/wrapper/select/shard/v3 v3.0.0-20210403073940-e7a7e3a05092
	github.com/asim/go-micro/v3 v3.6.1-0.20210831143116-05a299b76c7c
	github.com/gin-gonic/gin v1.7.0
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/golang/protobuf v1.5.2
	github.com/grpc-ecosystem/grpc-gateway v1.16.0
	github.com/pborman/uuid v1.2.1
	github.com/urfave/cli/v2 v2.3.0
	golang.org/x/net v0.0.0-20210510120150-4163338589ed
	google.golang.org/genproto v0.0.0-20210624195500-8bfb893ecb84
	google.golang.org/grpc v1.38.0
	google.golang.org/protobuf v1.26.0
)
