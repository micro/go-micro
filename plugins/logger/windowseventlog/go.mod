module github.com/asim/go-micro/plugins/logger/windowseventlog

go 1.17

require (
	github.com/stretchr/testify v1.7.0
	go-micro.dev/v4 v4.2.1
	golang.org/x/sys v0.0.0-20210616094352-59db8d763f22
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/google/uuid v1.2.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
)

replace go-micro.dev/v4 => ../../../../go-micro
