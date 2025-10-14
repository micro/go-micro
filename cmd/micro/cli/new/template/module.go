package template

var (
	Module = `module {{.Dir}}

go 1.18

require (
	go-micro.dev/v5 latest
	github.com/golang/protobuf latest
	google.golang.org/protobuf latest
)
`
)
