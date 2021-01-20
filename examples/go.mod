module github.com/asim/go-micro/examples/v3

go 1.13

replace k8s.io/api => k8s.io/api v0.0.0-20190708174958-539a33f6e817

replace k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190404173353-6a84e37a896d

replace k8s.io/apiserver => k8s.io/apiserver v0.0.0-20190708180123-608cd7da68f7

replace k8s.io/client-go => k8s.io/client-go v11.0.0+incompatible

replace k8s.io/component-base => k8s.io/component-base v0.0.0-20190708175518-244289f83105

replace google.golang.org/grpc => google.golang.org/grpc v1.24.0

require (
	github.com/99designs/gqlgen v0.10.1
	github.com/asim/go-micro/plugins/config/encoder/toml/v3 v3.0.0-20210120210110-dc8236ec05ed
	github.com/asim/go-micro/plugins/config/source/configmap/v3 v3.0.0-20210120210110-dc8236ec05ed
	github.com/asim/go-micro/plugins/config/source/grpc/v3 v3.0.0-20210120210110-dc8236ec05ed
	github.com/asim/go-micro/plugins/registry/etcd/v3 v3.0.0-20210120210110-dc8236ec05ed
	github.com/asim/go-micro/plugins/registry/kubernetes/v3 v3.0.0-20210120210110-dc8236ec05ed
	github.com/asim/go-micro/plugins/selector/static/v3 v3.0.0-20210120210110-dc8236ec05ed
	github.com/asim/go-micro/plugins/wrapper/select/roundrobin/v3 v3.0.0-20210120210110-dc8236ec05ed
	github.com/asim/go-micro/plugins/wrapper/select/shard/v3 v3.0.0-20210120210110-dc8236ec05ed
	github.com/asim/go-micro/v3 v3.0.0-20210120135431-d94936f6c97c
	github.com/astaxie/beego v1.12.0
	github.com/emicklei/go-restful v2.11.1+incompatible
	github.com/gin-gonic/gin v1.4.0
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/golang/protobuf v1.4.2
	github.com/google/uuid v1.1.1
	github.com/gorilla/rpc v1.2.0
	github.com/gorilla/websocket v1.4.1
	github.com/grpc-ecosystem/grpc-gateway v1.12.1
	github.com/hailocab/go-geoindex v0.0.0-20160127134810-64631bfe9711
	github.com/micro/cli/v2 v2.1.2
	github.com/pborman/uuid v1.2.0
	github.com/shiena/ansicolor v0.0.0-20151119151921-a422bbe96644 // indirect
	github.com/vektah/gqlparser v1.2.0
	golang.org/x/net v0.0.0-20200707034311-ab3426394381
	google.golang.org/genproto v0.0.0-20191216164720-4f79533eabd1
	google.golang.org/grpc v1.26.0
)
