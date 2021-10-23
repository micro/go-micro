module github.com/asim/go-micro/plugins/config/encoder/cue/v4

go 1.16

require (
	cuelang.org/go v0.0.15
	github.com/ghodss/yaml v1.0.0
	github.com/stretchr/testify v1.7.0
	go-micro.dev/v4 v4.2.1
)

replace go-micro.dev/v4 => ../../../../../go-micro
