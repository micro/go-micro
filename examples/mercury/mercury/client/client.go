package main

import (
	"fmt"
	"time"

	hello "github.com/micro/micro/examples/greeter/server/proto/hello"
	"github.com/mondough/mercury"
	tmsg "github.com/mondough/typhon/message"
	"github.com/mondough/typhon/rabbit"
)

func main() {
	req := mercury.NewRequest()
	req.SetService("foo")
	req.SetEndpoint("Say.Hello")
	req.SetBody(&hello.Request{
		Name: "John",
	})
	tmsg.ProtoMarshaler().MarshalBody(req)
	trans := rabbit.NewTransport()
	rsp, err := trans.Send(req, time.Second)
	if err != nil {
		fmt.Println(err)
		return
	}
	tmsg.ProtoUnmarshaler(new(hello.Response)).UnmarshalPayload(rsp)

	fmt.Println(rsp.Body())
}
