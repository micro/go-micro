package mock

import (
	"testing"

	"github.com/micro/go-micro/errors"

	"golang.org/x/net/context"
)

func TestClient(t *testing.T) {
	response := []MockResponse{
		{Method: "Foo.Bar", Response: map[string]interface{}{"foo": "bar"}},
		{Method: "Foo.Fail", Error: errors.InternalServerError("go.mock", "failed")},
	}

	c := NewClient(Response("go.mock", response))

	for _, r := range response {
		req := c.NewJsonRequest("go.mock", r.Method, map[string]interface{}{"foo": "bar"})
		var rsp map[string]interface{}

		err := c.Call(context.TODO(), req, &rsp)

		if err != r.Error {
			t.Fatalf("Expecter error %v got %v", r.Error, err)
		}

		t.Log(rsp)
	}

}
