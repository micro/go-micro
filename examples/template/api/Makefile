
GOPATH:=$(shell go env GOPATH)

.PHONY: proto test docker

proto:
	protoc --proto_path=${GOPATH}/src:. --micro_out=. --go_out=. proto/example/example.proto

build: proto
	go build -o template-api main.go
    
test:
	go test -v ./... -cover

docker:
	docker build . -t template-api:latest
