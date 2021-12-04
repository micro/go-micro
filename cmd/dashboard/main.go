package main

import (
	"github.com/asim/go-micro/cmd/dashboard/v4/config"
	"github.com/asim/go-micro/cmd/dashboard/v4/handler"
	mhttp "github.com/asim/go-micro/plugins/server/http/v4"
	"github.com/gin-gonic/gin"
	"go-micro.dev/v4"
	"go-micro.dev/v4/logger"
)

func main() {
	if err := config.Load(); err != nil {
		logger.Fatal(err)
	}
	srv := micro.NewService(micro.Server(mhttp.NewServer()))
	opts := []micro.Option{
		micro.Name(config.Name),
		micro.Address(config.GetServerConfig().Address),
		micro.Version(config.Version),
	}
	srv.Init(opts...)
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery(), gin.Logger())
	if err := handler.Register(handler.Options{Client: srv.Client(), Router: router}); err != nil {
		logger.Fatal(err)
	}
	if err := micro.RegisterHandler(srv.Server(), router); err != nil {
		logger.Fatal(err)
	}
	if err := srv.Run(); err != nil {
		logger.Fatal(err)
	}
}
