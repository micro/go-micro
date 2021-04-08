module github.com/asim/go-micro/examples/v3

go 1.13

replace k8s.io/api => k8s.io/api v0.0.0-20190708174958-539a33f6e817

replace k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190404173353-6a84e37a896d

replace k8s.io/apiserver => k8s.io/apiserver v0.0.0-20190708180123-608cd7da68f7

replace k8s.io/client-go => k8s.io/client-go v11.0.0+incompatible

replace k8s.io/component-base => k8s.io/component-base v0.0.0-20190708175518-244289f83105

replace google.golang.org/grpc => google.golang.org/grpc v1.24.0

require (
	github.com/asim/go-micro/plugins/client/http/v3 v3.0.0-20210403073940-e7a7e3a05092
	github.com/asim/go-micro/plugins/config/encoder/toml/v3 v3.0.0-20210403073940-e7a7e3a05092
	github.com/asim/go-micro/plugins/config/source/grpc/v3 v3.0.0-20210403073940-e7a7e3a05092
	github.com/asim/go-micro/plugins/server/http/v3 v3.0.0-20210403073940-e7a7e3a05092
	github.com/asim/go-micro/plugins/wrapper/select/roundrobin/v3 v3.0.0-20210403073940-e7a7e3a05092
	github.com/asim/go-micro/plugins/wrapper/select/shard/v3 v3.0.0-20210403073940-e7a7e3a05092
	github.com/asim/go-micro/v3 v3.5.0
	github.com/gin-gonic/gin v1.6.3
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/golang/protobuf v1.5.2
	github.com/grpc-ecosystem/grpc-gateway v1.16.0
	github.com/micro/cli/v2 v2.1.2
	github.com/pborman/uuid v1.2.1
	golang.org/x/net v0.0.0-20210405180319-a5a99cb37ef4
	google.golang.org/genproto v0.0.0-20210406143921-e86de6bf7a46
	google.golang.org/grpc v1.36.1
)
