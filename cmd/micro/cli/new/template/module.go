package template

var (
	Module = `module {{.Dir}}

go 1.22

require (
	go-micro.dev/v6 latest
	github.com/golang/protobuf latest
	google.golang.org/protobuf latest
)
`

	// ModuleNoProto is the default go.mod: no protobuf dependencies.
	ModuleNoProto = `module {{.Dir}}

go 1.22

require go-micro.dev/v6 latest
`
)
