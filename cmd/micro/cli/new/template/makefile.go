package template

var (
	Makefile = `
GOPATH:=$(shell go env GOPATH)
.PHONY: init
init:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install go-micro.dev/v5/cmd/protoc-gen-micro@latest
	go install github.com/google/gnostic/cmd/protoc-gen-openapi@latest

.PHONY: api
api:
	protoc --openapi_out=. --proto_path=. proto/{{.Alias}}.proto

.PHONY: proto
proto:
	protoc --proto_path=. --micro_out=. --go_out=:. proto/{{.Alias}}.proto
	
.PHONY: build
build:
	go build -o {{.Alias}} *.go

.PHONY: test
test:
	go test -v ./... -cover

.PHONY: docker
docker:
	docker build . -t {{.Alias}}:latest
`
)
