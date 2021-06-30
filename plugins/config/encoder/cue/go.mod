module github.com/asim/go-micro/plugins/config/encoder/cue/v3

go 1.16

require (
	cuelang.org/go v0.0.15
	github.com/asim/go-micro/v3 v3.5.2-0.20210630062103-c13bb07171bc
	github.com/ghodss/yaml v1.0.0
	github.com/stretchr/testify v1.7.0
)

replace github.com/asim/go-micro/v3 => ../../../../../go-micro
