
GOPATH:=$(shell go env GOPATH)

.PHONY: proto test docker


build:

	go build -o template-web main.go
    
test:
	go test -v ./... -cover

docker:
	docker build . -t template-web:latest
