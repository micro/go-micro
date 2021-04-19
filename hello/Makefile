
GOPATH:=$(shell go env GOPATH)
MODIFY=Mproto/imports/api.proto=github.com/micro/go-micro/v2/api/proto

## 可能存在的问题就是 go op 未设置
.PHONY: proto
proto:
	protoc --proto_path=. --micro_out=${MODIFY}:. --go_out=${MODIFY}:. proto/hello/hello.proto

.PHONY: build
build: proto
	go build -o hello-service *.go

.PHONY: test
test:
	go test -v ./... -cover