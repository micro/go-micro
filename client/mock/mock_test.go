package mock

import (
	"context"
	"testing"

	"github.com/micro/go-micro/errors"
)

func TestClient(t *testing.T) {
	type TestResponse struct {
		Param string
	}

	response := []MockResponse{
		{Endpoint: "Foo.Bar", Response: map[string]interface{}{"foo": "bar"}},
		{Endpoint: "Foo.Struct", Response: &TestResponse{Param: "aparam"}},
		{Endpoint: "Foo.Fail", Error: errors.InternalServerError("go.mock", "failed")},
		{Endpoint: "Foo.Func", Response: func() string { return "string" }},
		{Endpoint: "Foo.FuncStruct", Response: func() *TestResponse { return &TestResponse{Param: "aparam"} }},
	}

	c := NewClient(Response("go.mock", response))

	for _, r := range response {
		req := c.NewRequest("go.mock", r.Endpoint, map[string]interface{}{"foo": "bar"})
		var rsp interface{}

		err := c.Call(context.TODO(), req, &rsp)

		if err != r.Error {
			t.Fatalf("Expecter error %v got %v", r.Error, err)
		}

		t.Log(rsp)
	}

}
