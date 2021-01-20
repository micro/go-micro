// +build main4

package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gorilla/rpc"
	"github.com/gorilla/rpc/json"
	"github.com/asim/go-micro/v3/registry"
	"github.com/pborman/uuid"
)

var (
	service = &registry.Service{
		Name: "go.micro.srv.greeter",
		Nodes: []*registry.Node{
			{
				Id:      "go.micro.srv.greeter-" + uuid.NewUUID().String(),
				Address: "localhost",
				Port:    4000,
			},
		},
	}
)

type Say struct{}

type Request map[string]interface{}
type Response string

func (s *Say) Hello(r *http.Request, req *Request, rsp *Response) error {
	*rsp = Response(fmt.Sprintf("Hello %s!", (*req)["name"]))
	return nil
}

func main() {
	l, err := net.Listen("tcp", "localhost:4000")
	if err != nil {
		fmt.Println(err)
	}

	s := rpc.NewServer()
	s.RegisterCodec(json.NewCodec(), "application/json")
	s.RegisterService(new(Say), "")
	http.Handle("/", s)
	go http.Serve(l, http.DefaultServeMux)

	register(service)

	notify := make(chan os.Signal, 1)
	signal.Notify(notify, syscall.SIGTERM, syscall.SIGINT, syscall.SIGKILL)
	<-notify

	deregister(service)
	l.Close()
}
