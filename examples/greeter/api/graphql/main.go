package main

import (
	"github.com/99designs/gqlgen/handler"
	gql "github.com/micro/go-micro/examples/greeter/api/graphql/graphql"
	helloProto "github.com/micro/go-micro/examples/greeter/srv/proto/hello"
	"github.com/micro/go-micro/v2/client"
	"github.com/micro/go-micro/v2/util/log"
	"github.com/micro/go-micro/v2/web"
)

func main() {
	// create new web service
	service := web.NewService(
		web.Name("go.micro.api.greeter"),
		web.Version("latest"),
		web.Address(":8085"),
	)

	// initialise service
	if err := service.Init(); err != nil {
		log.Fatal(err)
	}

	// RPC client
	cl := helloProto.NewSayService("go.micro.srv.greeter", client.DefaultClient)

	// register graphql handlers
	service.Handle("/", handler.Playground("GraphQL playground", "/query"))
	service.Handle("/query", handler.GraphQL(gql.NewExecutableSchema(gql.Config{Resolvers: &gql.Resolver{Client: cl}})))

	// run service
	if err := service.Run(); err != nil {
		log.Fatal(err)

	}
}
