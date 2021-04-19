package main

import (
	"context"
	"fmt"
	"github.com/micro/go-micro/v2"
	"github.com/micro/go-micro/v2/client"
	grpcclient "github.com/micro/go-micro/v2/client/grpc"
	"github.com/micro/go-micro/v2/registry/etcd"
	grpcserver "github.com/micro/go-micro/v2/server/grpc"
	"hello/proto/hello"
	"time"
)

func main() {
	// New Service
	service := micro.NewService(
		micro.Client(grpcclient.NewClient()), // 如果需要修改默认的client 和 默认的service一定要写在最前面
		micro.Server(grpcserver.NewServer()),

		micro.Registry(etcd.NewRegistry()),  // 修改注册中心
		micro.Name("go.micro.client.hello"), // 服务名称
		micro.Version("latest"),             // 版本号
	)

	// server init
	service.Init()

	// create a new hello service client
	cli := hello.NewHelloService("go.micro.service.hello", service.Client())

	// call the endpoint hello.Call
	rsp, err := cli.Call(context.Background(), &hello.Request{Name: "Alice"}, func(options *client.CallOptions) {
		options.RequestTimeout = time.Second * 1
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(rsp.Msg)
}
