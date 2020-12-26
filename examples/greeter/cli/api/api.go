package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/golang/protobuf/proto"
	api "github.com/micro/micro/v2/api/proto"
)

func main() {

	req, err := proto.Marshal(&api.Request{Get: map[string]*api.Pair{"name": {Key: "name", Values: []string{"John"}}}})
	if err != nil {
		fmt.Println(err)
		return
	}

	r, err := http.Post("http://localhost:8080/greeter/say/hello", "application/protobuf", bytes.NewReader(req))
	if err != nil {
		fmt.Println(err)
		return
	}
	defer r.Body.Close()

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fmt.Println(err)
		return
	}

	rsp := &api.Response{}
	if err := proto.Unmarshal(b, rsp); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(rsp.Body)
}
