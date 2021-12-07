module github.com/asim/go-micro/cmd/dashboard/v4

go 1.17

require (
	github.com/asim/go-micro/plugins/broker/kafka/v4 v4.0.0-20211201082631-1e4dd94b71f1
	github.com/asim/go-micro/plugins/broker/mqtt/v4 v4.0.0-20211201082631-1e4dd94b71f1
	github.com/asim/go-micro/plugins/broker/nats/v4 v4.0.0-20211201082631-1e4dd94b71f1
	github.com/asim/go-micro/plugins/broker/rabbitmq/v4 v4.0.0-20211201082631-1e4dd94b71f1
	github.com/asim/go-micro/plugins/broker/redis/v4 v4.0.0-20211201082631-1e4dd94b71f1
	github.com/asim/go-micro/plugins/client/grpc/v4 v4.0.0-20211202051538-c0e0b2b22508
	github.com/asim/go-micro/plugins/client/http/v4 v4.0.0-20211201082631-1e4dd94b71f1
	github.com/asim/go-micro/plugins/client/mucp/v4 v4.0.0-20211201082631-1e4dd94b71f1
	github.com/asim/go-micro/plugins/config/encoder/toml/v4 v4.0.0-20211201082631-1e4dd94b71f1
	github.com/asim/go-micro/plugins/config/encoder/yaml/v4 v4.0.0-20211201082631-1e4dd94b71f1
	github.com/asim/go-micro/plugins/registry/consul/v4 v4.0.0-20211201082631-1e4dd94b71f1
	github.com/asim/go-micro/plugins/registry/etcd/v4 v4.0.0-20211202051538-c0e0b2b22508
	github.com/asim/go-micro/plugins/registry/eureka/v4 v4.0.0-20211201082631-1e4dd94b71f1
	github.com/asim/go-micro/plugins/registry/gossip/v4 v4.0.0-20211201082631-1e4dd94b71f1
	github.com/asim/go-micro/plugins/registry/kubernetes/v4 v4.0.0-20211205115617-97f169c424fe
	github.com/asim/go-micro/plugins/registry/nacos/v4 v4.0.0-20211201082631-1e4dd94b71f1
	github.com/asim/go-micro/plugins/registry/nats/v4 v4.0.0-20211201082631-1e4dd94b71f1
	github.com/asim/go-micro/plugins/registry/zookeeper/v4 v4.0.0-20211201082631-1e4dd94b71f1
	github.com/asim/go-micro/plugins/server/http/v4 v4.0.0-20211201082631-1e4dd94b71f1
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/gin-gonic/gin v1.7.7
	github.com/pkg/errors v0.9.1
	go-micro.dev/v4 v4.4.0
	golang.org/x/net v0.0.0-20210614182718-04defd469f4e
)

require (
	github.com/BurntSushi/toml v0.3.1 // indirect
	github.com/Microsoft/go-winio v0.5.0 // indirect
	github.com/ProtonMail/go-crypto v0.0.0-20210428141323-04723f9f07d7 // indirect
	github.com/Shopify/sarama v1.29.1 // indirect
	github.com/acomagu/bufpipe v1.0.3 // indirect
	github.com/aliyun/alibaba-cloud-sdk-go v1.61.976 // indirect
	github.com/armon/go-metrics v0.0.0-20180917152333-f0300d1749da // indirect
	github.com/bitly/go-simplejson v0.5.0 // indirect
	github.com/buger/jsonparser v0.0.0-20181115193947-bf1c66bbce23 // indirect
	github.com/cenkalti/backoff v2.2.1+incompatible // indirect
	github.com/clbanning/x2j v0.0.0-20191024224557-825249438eec // indirect
	github.com/coreos/go-semver v0.3.0 // indirect
	github.com/coreos/go-systemd/v22 v22.3.2 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/eapache/go-resiliency v1.2.0 // indirect
	github.com/eapache/go-xerial-snappy v0.0.0-20180814174437-776d5712da21 // indirect
	github.com/eapache/queue v1.1.0 // indirect
	github.com/eclipse/paho.mqtt.golang v1.3.5 // indirect
	github.com/emirpasic/gods v1.12.0 // indirect
	github.com/fatih/color v1.9.0 // indirect
	github.com/franela/goreq v0.0.0-20171204163338-bcd34c9993f8 // indirect
	github.com/fsnotify/fsnotify v1.4.9 // indirect
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/gin-contrib/sse v0.1.0 // indirect
	github.com/go-errors/errors v1.0.1 // indirect
	github.com/go-git/gcfg v1.5.0 // indirect
	github.com/go-git/go-billy/v5 v5.3.1 // indirect
	github.com/go-git/go-git/v5 v5.4.2 // indirect
	github.com/go-playground/locales v0.13.0 // indirect
	github.com/go-playground/universal-translator v0.17.0 // indirect
	github.com/go-playground/validator/v10 v10.4.1 // indirect
	github.com/go-zookeeper/zk v1.0.2 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/golang/snappy v0.0.3 // indirect
	github.com/gomodule/redigo v1.8.5 // indirect
	github.com/google/btree v1.0.0 // indirect
	github.com/google/uuid v1.2.0 // indirect
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/hashicorp/consul/api v1.9.0 // indirect
	github.com/hashicorp/errwrap v1.0.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.1 // indirect
	github.com/hashicorp/go-hclog v0.12.0 // indirect
	github.com/hashicorp/go-immutable-radix v1.0.0 // indirect
	github.com/hashicorp/go-msgpack v0.5.3 // indirect
	github.com/hashicorp/go-multierror v1.1.0 // indirect
	github.com/hashicorp/go-rootcerts v1.0.2 // indirect
	github.com/hashicorp/go-sockaddr v1.0.0 // indirect
	github.com/hashicorp/go-uuid v1.0.2 // indirect
	github.com/hashicorp/golang-lru v0.5.1 // indirect
	github.com/hashicorp/memberlist v0.2.2 // indirect
	github.com/hashicorp/serf v0.9.5 // indirect
	github.com/hudl/fargo v1.3.0 // indirect
	github.com/imdario/mergo v0.3.12 // indirect
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99 // indirect
	github.com/jcmturner/aescts/v2 v2.0.0 // indirect
	github.com/jcmturner/dnsutils/v2 v2.0.0 // indirect
	github.com/jcmturner/gofork v1.0.0 // indirect
	github.com/jcmturner/gokrb5/v8 v8.4.2 // indirect
	github.com/jcmturner/rpc/v2 v2.0.3 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/json-iterator/go v1.1.11 // indirect
	github.com/kevinburke/ssh_config v0.0.0-20201106050909-4977a11b4351 // indirect
	github.com/klauspost/compress v1.12.2 // indirect
	github.com/leodido/go-urn v1.2.0 // indirect
	github.com/lestrrat/go-file-rotatelogs v0.0.0-20180223000712-d3151e2a480f // indirect
	github.com/lestrrat/go-strftime v0.0.0-20180220042222-ba3bf9c1d042 // indirect
	github.com/mattn/go-colorable v0.1.8 // indirect
	github.com/mattn/go-isatty v0.0.12 // indirect
	github.com/miekg/dns v1.1.43 // indirect
	github.com/minio/highwayhash v1.0.2 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/hashstructure v1.1.0 // indirect
	github.com/mitchellh/mapstructure v1.3.3 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.1 // indirect
	github.com/nacos-group/nacos-sdk-go/v2 v2.0.0-Beta.1 // indirect
	github.com/nats-io/jwt/v2 v2.2.0 // indirect
	github.com/nats-io/nats.go v1.11.1-0.20210623165838-4b75fc59ae30 // indirect
	github.com/nats-io/nkeys v0.3.0 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/nxadm/tail v1.4.8 // indirect
	github.com/op/go-logging v0.0.0-20160315200505-970db520ece7 // indirect
	github.com/oxtoacart/bpool v0.0.0-20190530202638-03653db5a59c // indirect
	github.com/patrickmn/go-cache v2.1.0+incompatible // indirect
	github.com/pierrec/lz4 v2.6.0+incompatible // indirect
	github.com/rcrowley/go-metrics v0.0.0-20201227073835-cf1acfcdf475 // indirect
	github.com/russross/blackfriday/v2 v2.0.1 // indirect
	github.com/sean-/seed v0.0.0-20170313163322-e2103e2c3529 // indirect
	github.com/sergi/go-diff v1.1.0 // indirect
	github.com/shurcooL/sanitized_anchor_name v1.0.0 // indirect
	github.com/streadway/amqp v1.0.0 // indirect
	github.com/toolkits/concurrent v0.0.0-20150624120057-a4371d70e3e3 // indirect
	github.com/ugorji/go/codec v1.1.7 // indirect
	github.com/urfave/cli/v2 v2.3.0 // indirect
	github.com/xanzy/ssh-agent v0.3.0 // indirect
	go.etcd.io/etcd/api/v3 v3.5.0 // indirect
	go.etcd.io/etcd/client/pkg/v3 v3.5.0 // indirect
	go.etcd.io/etcd/client/v3 v3.5.0 // indirect
	go.uber.org/atomic v1.7.0 // indirect
	go.uber.org/multierr v1.6.0 // indirect
	go.uber.org/zap v1.17.0 // indirect
	golang.org/x/crypto v0.0.0-20210616213533-5ff15b29337e // indirect
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d // indirect
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c // indirect
	golang.org/x/sys v0.0.0-20210615035016-665e8c7367d1 // indirect
	golang.org/x/text v0.3.6 // indirect
	google.golang.org/appengine v1.6.5 // indirect
	google.golang.org/genproto v0.0.0-20210602131652-f16073e35f0c // indirect
	google.golang.org/grpc v1.42.0 // indirect
	google.golang.org/protobuf v1.27.1 // indirect
	gopkg.in/gcfg.v1 v1.2.3 // indirect
	gopkg.in/ini.v1 v1.62.0 // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)
