module github.com/asim/go-micro/plugins/config/encoder/yaml/v4

go 1.17

require (
	github.com/ghodss/yaml v1.0.0
	go-micro.dev/v4 v4.2.1
)

require (
	github.com/kr/text v0.2.0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)

replace go-micro.dev/v4 => ../../../../../go-micro
