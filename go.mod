module github.com/micro/go-micro/v3

go 1.13

replace github.com/imdario/mergo => github.com/imdario/mergo v0.3.8

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/bitly/go-simplejson v0.5.0
	github.com/bmizerany/assert v0.0.0-20160611221934-b7ed37b82869 // indirect
	github.com/bradfitz/gomemcache v0.0.0-20190913173617-a41fca850d0b
	github.com/caddyserver/certmagic v0.10.6
	github.com/coreos/bbolt v1.3.3 // indirect
	github.com/coreos/etcd v3.3.18+incompatible
	github.com/coreos/go-semver v0.3.0 // indirect
	github.com/coreos/go-systemd v0.0.0-20190719114852-fd7a80b32e1f // indirect
	github.com/coreos/pkg v0.0.0-20180928190104-399ea9e2e55f // indirect
	github.com/davecgh/go-spew v1.1.1
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/ef-ds/deque v1.0.4-0.20190904040645-54cb57c252a1
	github.com/evanphx/json-patch/v5 v5.0.0
	github.com/fsnotify/fsnotify v1.4.7
	github.com/fsouza/go-dockerclient v1.6.0
	github.com/ghodss/yaml v1.0.0
	github.com/go-acme/lego/v3 v3.4.0
	github.com/go-git/go-git/v5 v5.1.0
	github.com/gobwas/httphead v0.0.0-20180130184737-2c6c146eadee
	github.com/gobwas/pool v0.2.0 // indirect
	github.com/gobwas/ws v1.0.3
	github.com/golang/groupcache v0.0.0-20200121045136-8c9f03a8e57e // indirect
	github.com/golang/protobuf v1.4.2
	github.com/google/uuid v1.1.1
	github.com/gorilla/handlers v1.4.2
	github.com/gorilla/websocket v1.4.1 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.1.0 // indirect
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.9.5 // indirect
	github.com/hashicorp/hcl v1.0.0
	github.com/hpcloud/tail v1.0.0
	github.com/imdario/mergo v0.3.9
	github.com/jonboulle/clockwork v0.1.0 // indirect
	github.com/kr/pretty v0.2.0
	github.com/lib/pq v1.7.0
	github.com/lucas-clemente/quic-go v0.14.1
	github.com/miekg/dns v1.1.27
	github.com/mitchellh/hashstructure v1.0.0
	github.com/nats-io/nats-streaming-server v0.18.0 // indirect
	github.com/nats-io/nats.go v1.10.0
	github.com/nats-io/stan.go v0.7.0
	github.com/onsi/ginkgo v1.12.0 // indirect
	github.com/oxtoacart/bpool v0.0.0-20190530202638-03653db5a59c
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.7.0
	github.com/soheilhy/cmux v0.1.4 // indirect
	github.com/stretchr/testify v1.5.1
	github.com/teris-io/shortid v0.0.0-20171029131806-771a37caa5cf
	github.com/tmc/grpc-websocket-proxy v0.0.0-20200122045848-3419fae592fc // indirect
	github.com/xanzy/go-gitlab v0.35.1 // indirect
	github.com/xiang90/probing v0.0.0-20190116061207-43a291ad63a2 // indirect
	go.etcd.io/bbolt v1.3.5
	go.uber.org/zap v1.13.0
	golang.org/x/crypto v0.0.0-20200709230013-948cd5f35899
	golang.org/x/net v0.0.0-20200520182314-0ba52f642ac2
	golang.org/x/tools v0.0.0-20200117065230-39095c1d176c // indirect
	google.golang.org/genproto v0.0.0-20200526211855-cb27e3aa2013
	google.golang.org/grpc v1.27.0
	google.golang.org/protobuf v1.25.0
	gopkg.in/square/go-jose.v2 v2.4.1 // indirect
	gopkg.in/yaml.v2 v2.2.8 // indirect
	sigs.k8s.io/yaml v1.1.0 // indirect
)

replace google.golang.org/grpc => google.golang.org/grpc v1.26.0
