module github.com/asim/go-micro/examples/v4

go 1.17

replace (
	github.com/asim/go-micro/plugins/client/grpc/v4 => ../plugins/client/grpc
	github.com/asim/go-micro/plugins/client/http/v4 => ../plugins/client/http
	github.com/asim/go-micro/plugins/config/encoder/toml/v4 => ../plugins/config/encoder/toml
	github.com/asim/go-micro/plugins/config/encoder/yaml/v4 => ../plugins/config/encoder/yaml
	github.com/asim/go-micro/plugins/config/source/grpc/v4 => ../plugins/config/source/grpc
	github.com/asim/go-micro/plugins/server/grpc/v4 => ../plugins/server/grpc
	github.com/asim/go-micro/plugins/server/http/v4 => ../plugins/server/http
	github.com/asim/go-micro/plugins/transport/grpc/v4 => ../plugins/transport/grpc
	github.com/asim/go-micro/plugins/wrapper/select/roundrobin/v4 => ../plugins/wrapper/select/roundrobin
	github.com/asim/go-micro/plugins/wrapper/select/shard/v4 => ../plugins/wrapper/select/shard
	go-micro.dev/v4 => ../../go-micro
)

require (
	github.com/asim/go-micro/plugins/client/grpc/v4 v4.0.0-20211019191242-9edc569e68bb
	github.com/asim/go-micro/plugins/client/http/v4 v4.0.0-00010101000000-000000000000
	github.com/asim/go-micro/plugins/config/encoder/toml/v4 v4.0.0-00010101000000-000000000000
	github.com/asim/go-micro/plugins/config/encoder/yaml/v4 v4.0.0-00010101000000-000000000000
	github.com/asim/go-micro/plugins/config/source/grpc/v4 v4.0.0-00010101000000-000000000000
	github.com/asim/go-micro/plugins/server/grpc/v4 v4.0.0-00010101000000-000000000000
	github.com/asim/go-micro/plugins/server/http/v4 v4.0.0-00010101000000-000000000000
	github.com/asim/go-micro/plugins/wrapper/select/roundrobin/v4 v4.0.0-00010101000000-000000000000
	github.com/asim/go-micro/plugins/wrapper/select/shard/v4 v4.0.0-00010101000000-000000000000
	github.com/gin-gonic/gin v1.7.4
	github.com/golang/glog v1.0.0
	github.com/golang/protobuf v1.5.2
	github.com/grpc-ecosystem/grpc-gateway v1.16.0
	github.com/pborman/uuid v1.2.1
	github.com/urfave/cli/v2 v2.3.0
	go-micro.dev/v4 v4.3.0
	golang.org/x/net v0.0.0-20211020060615-d418f374d309
	google.golang.org/genproto v0.0.0-20211021150943-2b146023228c
	google.golang.org/grpc v1.42.0
	google.golang.org/protobuf v1.27.1
)

require (
	github.com/BurntSushi/toml v0.3.1 // indirect
	github.com/Microsoft/go-winio v0.5.0 // indirect
	github.com/ProtonMail/go-crypto v0.0.0-20210428141323-04723f9f07d7 // indirect
	github.com/acomagu/bufpipe v1.0.3 // indirect
	github.com/bitly/go-simplejson v0.5.0 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.0 // indirect
	github.com/emirpasic/gods v1.12.0 // indirect
	github.com/fsnotify/fsnotify v1.4.9 // indirect
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/gin-contrib/sse v0.1.0 // indirect
	github.com/go-git/gcfg v1.5.0 // indirect
	github.com/go-git/go-billy/v5 v5.3.1 // indirect
	github.com/go-git/go-git/v5 v5.4.2 // indirect
	github.com/go-playground/locales v0.13.0 // indirect
	github.com/go-playground/universal-translator v0.17.0 // indirect
	github.com/go-playground/validator/v10 v10.4.1 // indirect
	github.com/google/uuid v1.2.0 // indirect
	github.com/imdario/mergo v0.3.12 // indirect
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99 // indirect
	github.com/json-iterator/go v1.1.9 // indirect
	github.com/kevinburke/ssh_config v0.0.0-20201106050909-4977a11b4351 // indirect
	github.com/leodido/go-urn v1.2.0 // indirect
	github.com/mattn/go-isatty v0.0.12 // indirect
	github.com/miekg/dns v1.1.43 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.1 // indirect
	github.com/nxadm/tail v1.4.8 // indirect
	github.com/oxtoacart/bpool v0.0.0-20190530202638-03653db5a59c // indirect
	github.com/patrickmn/go-cache v2.1.0+incompatible // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/russross/blackfriday/v2 v2.0.1 // indirect
	github.com/sergi/go-diff v1.1.0 // indirect
	github.com/shurcooL/sanitized_anchor_name v1.0.0 // indirect
	github.com/ugorji/go/codec v1.1.7 // indirect
	github.com/xanzy/ssh-agent v0.3.0 // indirect
	golang.org/x/crypto v0.0.0-20210513164829-c07d793c2f9a // indirect
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c // indirect
	golang.org/x/sys v0.0.0-20210510120138-977fb7262007 // indirect
	golang.org/x/text v0.3.6 // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)
