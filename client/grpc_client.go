package client

import (
	"fmt"
	"github.com/asim/go-micro/registry"
	"math/rand"
	"net/http"
	"time"

	"github.com/asim/go-micro/errors"
	"google.golang.org/grpc"
)

type headerRoundTripper struct {
	r http.RoundTripper
}

type GRPCClient struct{}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func (r *GRPCClient) NewRequest(serviceName string, f RequestFunc) error {
	service, err := registry.GetService(serviceName)
	if err != nil {
		return errors.InternalServerError("go.micro.client", err.Error())
	}

	if len(service.Nodes()) == 0 {
		return errors.NotFound("go.micro.client", "Service not found")
	}

	n := rand.Int() % len(service.Nodes())
	node := service.Nodes()[n]
	address := fmt.Sprintf("%s:%d", node.Address(), node.Port())

	return f(address)
}

func NewGRPCClient() *GRPCClient {
	return &GRPCClient{}
}

func GRPCRequest(f func(cc *grpc.ClientConn) error) RequestFunc {
	return func(address string) error {
		fmt.Println(address)
		cc, err := grpc.Dial(address)
		if err != nil {
			return err
		}
		defer cc.Close()

		return f(cc)
	}
}
