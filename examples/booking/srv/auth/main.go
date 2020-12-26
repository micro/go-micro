package main

import (
	"encoding/json"
	"errors"
	"log"

	"github.com/micro/go-micro/examples/booking/data"
	"github.com/micro/go-micro/examples/booking/srv/auth/proto"

	"context"
	"golang.org/x/net/trace"

	"github.com/micro/go-micro/v2"
	"github.com/micro/go-micro/v2/metadata"
)

type Auth struct {
	customers map[string]*auth.Customer
}

// VerifyToken returns a customer from authentication token.
func (s *Auth) VerifyToken(ctx context.Context, req *auth.Request, rsp *auth.Result) error {
	md, _ := metadata.FromContext(ctx)
	traceID := md["traceID"]

	if tr, ok := trace.FromContext(ctx); ok {
		tr.LazyPrintf("traceID %s", traceID)
	}

	customer := s.customers[req.AuthToken]
	if customer == nil {
		return errors.New("Invalid Token")
	}

	rsp.Customer = customer
	return nil
}

// loadCustomers loads customers from a JSON file.
func loadCustomerData(path string) map[string]*auth.Customer {
	file := data.MustAsset(path)
	customers := []*auth.Customer{}

	// unmarshal JSON
	if err := json.Unmarshal(file, &customers); err != nil {
		log.Fatalf("Failed to unmarshal json: %v", err)
	}

	// create customer lookup map
	cache := make(map[string]*auth.Customer)
	for _, c := range customers {
		cache[c.AuthToken] = c
	}
	return cache
}

func main() {
	service := micro.NewService(
		micro.Name("go.micro.srv.auth"),
	)

	service.Init()

	auth.RegisterAuthHandler(service.Server(), &Auth{
		customers: loadCustomerData("data/customers.json"),
	})

	service.Run()
}
