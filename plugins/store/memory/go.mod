module github.com/asim/go-micro/plugins/store/memory/v4

go 1.16

require (
	github.com/kr/pretty v0.2.1
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/pkg/errors v0.9.1
	go-micro.dev/v4 v4.1.0
)

replace go-micro.dev/v4 => ../../../../go-micro
