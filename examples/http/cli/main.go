package main

import (
	"context"
	"log"

	httpClient "github.com/chinahtl/go-micro/plugins/client/http/v3"
	"github.com/chinahtl/go-micro/v3/client"
	"github.com/chinahtl/go-micro/v3/registry"
	"github.com/chinahtl/go-micro/v3/selector"
)

func main() {
	CallHttpServer()
}

func CallHttpServer() {
	r := registry.NewRegistry()
	s := selector.NewSelector(selector.Registry(r))
	// new client
	c := httpClient.NewClient(client.Selector(s))
	// create request/response
	request := c.NewRequest("demo-http", "/demo", "", client.WithContentType("application/json"))

	response := new(map[string]interface{})
	// call service
	err := c.Call(context.Background(), request, response)
	log.Printf("err:%v response:%#v\n", err, response)

}
