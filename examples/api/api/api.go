package main

import (
	"context"
	"encoding/json"
	"log"
	"strings"

	proto "github.com/micro/go-micro/examples/api/api/proto"
	"github.com/micro/go-micro/v2"
	api "github.com/micro/go-micro/v2/api/proto"
	"github.com/micro/go-micro/v2/errors"
)

type Example struct{}

type Foo struct{}

// Example.Call is a method which will be served by http request /example/call
// In the event we see /[service]/[method] the [service] is used as part of the method
// E.g /example/call goes to go.micro.api.example Example.Call
func (e *Example) Call(ctx context.Context, req *api.Request, rsp *api.Response) error {
	log.Print("Received Example.Call request")

	// parse values from the get request
	name, ok := req.Get["name"]

	if !ok || len(name.Values) == 0 {
		return errors.BadRequest("go.micro.api.example", "no content")
	}

	// set response status
	rsp.StatusCode = 200

	// respond with some json
	b, _ := json.Marshal(map[string]string{
		"message": "got your request " + strings.Join(name.Values, " "),
	})

	// set json body
	rsp.Body = string(b)

	return nil
}

// Foo.Bar is a method which will be served by http request /example/foo/bar
// Because Foo is not the same as the service name it is mapped beyond /example/
func (f *Foo) Bar(ctx context.Context, req *api.Request, rsp *api.Response) error {
	log.Print("Received Foo.Bar request")

	// check method
	if req.Method != "POST" {
		return errors.BadRequest("go.micro.api.example", "require post")
	}

	// let's make sure we get json
	ct, ok := req.Header["Content-Type"]
	if !ok || len(ct.Values) == 0 {
		return errors.BadRequest("go.micro.api.example", "need content-type")
	}

	if ct.Values[0] != "application/json" {
		return errors.BadRequest("go.micro.api.example", "expect application/json")
	}

	// parse body
	var body map[string]interface{}
	json.Unmarshal([]byte(req.Body), &body)

	// do something with parsed body

	return nil
}

func main() {
	service := micro.NewService(
		micro.Name("go.micro.api.example"),
	)

	service.Init()

	// register example handler
	proto.RegisterExampleHandler(service.Server(), new(Example))

	// register foo handler
	proto.RegisterFooHandler(service.Server(), new(Foo))

	if err := service.Run(); err != nil {
		log.Fatal(err)
	}
}
