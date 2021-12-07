package handler

import (
	"github.com/asim/go-micro/cmd/dashboard/v4/config"
	"github.com/asim/go-micro/cmd/dashboard/v4/handler/account"
	handlerclient "github.com/asim/go-micro/cmd/dashboard/v4/handler/client"
	"github.com/asim/go-micro/cmd/dashboard/v4/handler/registry"
	"github.com/asim/go-micro/cmd/dashboard/v4/handler/route"
	"github.com/asim/go-micro/cmd/dashboard/v4/handler/statistics"
	"github.com/asim/go-micro/cmd/dashboard/v4/web"
	"github.com/gin-gonic/gin"
	"go-micro.dev/v4/client"
)

type Options struct {
	Client client.Client
	Router *gin.Engine
}

func Register(opts Options) error {
	router := opts.Router
	if err := web.RegisterRoute(router); err != nil {
		return err
	}
	if cfg := config.GetServerConfig().CORS; cfg.Enable {
		router.Use(route.CorsHandler(cfg.Origin))
	}
	for _, r := range []route.Registrar{
		account.NewRouteRegistrar(),
		handlerclient.NewRouteRegistrar(opts.Client, opts.Client.Options().Registry),
		registry.NewRouteRegistrar(opts.Client.Options().Registry),
		statistics.NewRouteRegistrar(opts.Client.Options().Registry),
	} {
		r.RegisterRoute(router.Group(""))
	}
	return nil
}
