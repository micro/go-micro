module github.com/micro/go-micro

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/beevik/ntp v0.2.0
	github.com/bitly/go-simplejson v0.5.0
	github.com/bradfitz/gomemcache v0.0.0-20190329173943-551aad21a668
	github.com/bwmarrin/discordgo v0.19.0
	github.com/coreos/etcd v3.3.13+incompatible
	github.com/forestgiant/sliceutil v0.0.0-20160425183142-94783f95db6c
	github.com/fsnotify/fsnotify v1.4.7
	github.com/fsouza/go-dockerclient v1.4.1
	github.com/ghodss/yaml v1.0.0
	github.com/go-log/log v0.1.0
	github.com/go-redsync/redsync v1.2.0
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/golang/protobuf v1.3.1
	github.com/gomodule/redigo v2.0.0+incompatible
	github.com/google/uuid v1.1.1
	github.com/gorilla/handlers v1.4.0
	github.com/gorilla/websocket v1.4.0
	github.com/grpc-ecosystem/grpc-gateway v1.9.0
	github.com/hashicorp/consul v1.5.1
	github.com/hashicorp/consul/api v1.1.0
	github.com/hashicorp/hcl v1.0.0
	github.com/hashicorp/memberlist v0.1.4
	github.com/imdario/mergo v0.3.7
	github.com/joncalhoun/qson v0.0.0-20170526102502-8a9cab3a62b1
	github.com/json-iterator/go v1.1.6
	github.com/micro/cli v0.2.0
	github.com/micro/examples v0.1.0
	github.com/micro/go-plugins v1.1.0
	github.com/micro/mdns v0.1.0
	github.com/micro/micro v1.3.0
	github.com/mitchellh/hashstructure v1.0.0
	github.com/nats-io/nats.go v1.7.2
	github.com/nlopes/slack v0.5.0
	github.com/pborman/uuid v1.2.0
	github.com/pkg/errors v0.8.1
	go.etcd.io/etcd v3.3.13+incompatible
	golang.org/x/crypto v0.0.0-20190530122614-20be4c3c3ed5
	golang.org/x/mod v0.1.0 // indirect
	golang.org/x/net v0.0.0-20190603091049-60506f45cf65
	golang.org/x/sys v0.0.0-20190602015325-4c4f7f33c9ed // indirect
	golang.org/x/tools v0.0.0-20190603152906-08e0b306e832 // indirect
	google.golang.org/genproto v0.0.0-20190530194941-fb225487d101
	google.golang.org/grpc v1.21.0
	gopkg.in/go-playground/validator.v9 v9.29.0
	gopkg.in/redis.v3 v3.6.4
	gopkg.in/src-d/go-git.v4 v4.11.0
	gopkg.in/telegram-bot-api.v4 v4.6.4
	honnef.co/go/tools v0.0.0-20190602125119-5a4a2f4a438d // indirect
)

exclude sourcegraph.com/sourcegraph/go-diff v0.5.1

replace (
	github.com/golang/lint => github.com/golang/lint v0.0.0-20190227174305-8f45f776aaf1
	github.com/testcontainers/testcontainer-go => github.com/testcontainers/testcontainers-go v0.0.0-20181115231424-8e868ca12c0f
)
