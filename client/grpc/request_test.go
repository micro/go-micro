package grpc

import (
	"testing"
)

func TestMethodToGRPC(t *testing.T) {
	testData := []struct {
		service string
		method  string
		expect  string
	}{
		{
			"helloworld",
			"Greeter.SayHello",
			"/helloworld.Greeter/SayHello",
		},
		{
			"helloworld",
			"/helloworld.Greeter/SayHello",
			"/helloworld.Greeter/SayHello",
		},
		{
			"",
			"/helloworld.Greeter/SayHello",
			"/helloworld.Greeter/SayHello",
		},
		{
			"",
			"Greeter.SayHello",
			"/Greeter/SayHello",
		},
	}

	for _, d := range testData {
		method := methodToGRPC(d.service, d.method)
		if method != d.expect {
			t.Fatalf("expected %s got %s", d.expect, method)
		}
	}
}
