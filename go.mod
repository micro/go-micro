module github.com/micro/go-micro

exclude (
	github.com/Sirupsen/logrus v1.1.0
	github.com/Sirupsen/logrus v1.1.1
	github.com/Sirupsen/logrus v1.2.0
	github.com/Sirupsen/logrus v1.3.0
	github.com/Sirupsen/logrus v1.4.0
	github.com/Sirupsen/logrus v1.4.1
	github.com/Sirupsen/logrus v1.4.2
	sourcegraph.com/sourcegraph/go-diff v0.5.1
)

replace (
	github.com/golang/lint => github.com/golang/lint v0.0.0-20190227174305-8f45f776aaf1
	github.com/nats-io/go-nats => github.com/nats-io/nats.go v1.8.1
	github.com/nats-io/go-nats-streaming => github.com/nats-io/stan.go v0.5.0
	github.com/testcontainers/testcontainer-go => github.com/testcontainers/testcontainers-go v0.0.0-20181115231424-8e868ca12c0f
)

require (
	cloud.google.com/go v0.39.0 // indirect
	github.com/BurntSushi/toml v0.3.1
	github.com/OneOfOne/xxhash v1.2.5 // indirect
	github.com/armon/circbuf v0.0.0-20190214190532-5111143e8da2 // indirect
	github.com/armon/go-metrics v0.0.0-20190430140413-ec5e00d3c878 // indirect
	github.com/armon/go-radix v1.0.0 // indirect
	github.com/beevik/ntp v0.2.0
	github.com/bitly/go-simplejson v0.5.0
	github.com/bradfitz/gomemcache v0.0.0-20190329173943-551aad21a668
	github.com/bwmarrin/discordgo v0.19.0
	github.com/containerd/continuity v0.0.0-20190426062206-aaeac12a7ffc // indirect
	github.com/coreos/etcd v3.3.13+incompatible
	github.com/dgryski/go-sip13 v0.0.0-20190329191031-25c5027a8c7b // indirect
	github.com/emirpasic/gods v1.12.0 // indirect
	github.com/forestgiant/sliceutil v0.0.0-20160425183142-94783f95db6c
	github.com/fsnotify/fsnotify v1.4.7
	github.com/fsouza/go-dockerclient v1.4.1
	github.com/ghodss/yaml v1.0.0
	github.com/gliderlabs/ssh v0.1.4 // indirect
	github.com/go-log/log v0.1.0
	github.com/go-playground/locales v0.12.1 // indirect
	github.com/go-playground/universal-translator v0.16.0 // indirect
	github.com/go-redsync/redsync v1.2.0
	github.com/golang/mock v1.3.1 // indirect
	github.com/golang/protobuf v1.3.1
	github.com/gomodule/redigo v2.0.0+incompatible
	github.com/google/btree v1.0.0 // indirect
	github.com/google/pprof v0.0.0-20190515194954-54271f7e092f // indirect
	github.com/google/uuid v1.1.1
	github.com/gorilla/handlers v1.4.0
	github.com/gorilla/websocket v1.4.0
	github.com/hashicorp/consul/api v1.1.0
	github.com/hashicorp/go-immutable-radix v1.1.0 // indirect
	github.com/hashicorp/go-msgpack v0.5.5 // indirect
	github.com/hashicorp/go-sockaddr v1.0.2 // indirect
	github.com/hashicorp/go-version v1.2.0 // indirect
	github.com/hashicorp/hcl v1.0.0
	github.com/hashicorp/mdns v1.0.1 // indirect
	github.com/hashicorp/memberlist v0.1.4
	github.com/hashicorp/serf v0.8.3 // indirect
	github.com/imdario/mergo v0.3.7
	github.com/joncalhoun/qson v0.0.0-20170526102502-8a9cab3a62b1
	github.com/json-iterator/go v1.1.6
	github.com/kisielk/errcheck v1.2.0 // indirect
	github.com/konsorten/go-windows-terminal-sequences v1.0.2 // indirect
	github.com/kr/pty v1.1.4 // indirect
	github.com/leodido/go-urn v1.1.0 // indirect
	github.com/lucas-clemente/quic-go v0.11.2
	github.com/marten-seemann/qtls v0.2.4 // indirect
	github.com/mattn/go-colorable v0.1.2 // indirect
	github.com/micro/cli v0.2.0
	github.com/micro/mdns v0.1.0
	github.com/miekg/dns v1.1.13 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/gox v1.0.1 // indirect
	github.com/mitchellh/hashstructure v1.0.0
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.1 // indirect
	github.com/nats-io/nats.go v1.8.1
	github.com/nlopes/slack v0.5.0
	github.com/onsi/ginkgo v1.8.0 // indirect
	github.com/onsi/gomega v1.5.0 // indirect
	github.com/pborman/uuid v1.2.0
	github.com/pkg/errors v0.8.1
	github.com/posener/complete v1.2.1 // indirect
	github.com/prometheus/client_golang v0.9.3 // indirect
	github.com/prometheus/common v0.4.1 // indirect
	github.com/prometheus/procfs v0.0.2 // indirect
	github.com/prometheus/tsdb v0.8.0 // indirect
	github.com/sirupsen/logrus v1.4.2 // indirect
	github.com/spaolacci/murmur3 v1.1.0 // indirect
	github.com/stretchr/objx v0.2.0 // indirect
	github.com/technoweenie/multipartstreamer v1.0.1 // indirect
	github.com/xanzy/ssh-agent v0.2.1 // indirect
	go.etcd.io/etcd v3.3.13+incompatible
	go.opencensus.io v0.22.0 // indirect
	golang.org/x/crypto v0.0.0-20190530122614-20be4c3c3ed5
	golang.org/x/exp v0.0.0-20190510132918-efd6b22b2522 // indirect
	golang.org/x/image v0.0.0-20190523035834-f03afa92d3ff // indirect
	golang.org/x/lint v0.0.0-20190409202823-959b441ac422 // indirect
	golang.org/x/mobile v0.0.0-20190509164839-32b2708ab171 // indirect
	golang.org/x/mod v0.1.0 // indirect
	golang.org/x/net v0.0.0-20190603091049-60506f45cf65
	golang.org/x/oauth2 v0.0.0-20190604053449-0f29369cfe45 // indirect
	golang.org/x/sys v0.0.0-20190602015325-4c4f7f33c9ed // indirect
	golang.org/x/time v0.0.0-20190308202827-9d24e82272b4 // indirect
	golang.org/x/tools v0.0.0-20190603231351-8aaa1484dc10 // indirect
	google.golang.org/appengine v1.6.0 // indirect
	google.golang.org/genproto v0.0.0-20190530194941-fb225487d101 // indirect
	google.golang.org/grpc v1.21.1
	gopkg.in/bsm/ratelimit.v1 v1.0.0-20160220154919-db14e161995a // indirect
	gopkg.in/go-playground/validator.v9 v9.29.0
	gopkg.in/redis.v3 v3.6.4
	gopkg.in/src-d/go-billy.v4 v4.3.0 // indirect
	gopkg.in/src-d/go-git-fixtures.v3 v3.5.0 // indirect
	gopkg.in/src-d/go-git.v4 v4.11.0
	gopkg.in/telegram-bot-api.v4 v4.6.4
	honnef.co/go/tools v0.0.0-20190604153307-63e9ff576adb // indirect
)
