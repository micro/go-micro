package statistics

import (
	"github.com/asim/go-micro/cmd/dashboard/v4/config"
	"github.com/asim/go-micro/cmd/dashboard/v4/handler/route"
	"github.com/gin-gonic/gin"
	"go-micro.dev/v4/registry"
)

type service struct {
	registry registry.Registry
}

func NewRouteRegistrar(registry registry.Registry) route.Registrar {
	return service{registry: registry}
}

func (s service) RegisterRoute(router gin.IRoutes) {
	router.GET("/version", s.GetVersion)
	router.Use(route.AuthRequired()).GET("/api/summary", s.GetSummary)
}

// @Security ApiKeyAuth
// @Tags Statistics
// @ID statistics_getSummary
// @Success 200 	{object}	getSummaryResponse
// @Failure 400 	{object}	string
// @Failure 401 	{object}	string
// @Failure 500		{object}	string
// @Router /api/summary [get]
func (s *service) GetSummary(ctx *gin.Context) {
	services, err := s.registry.ListServices()
	if err != nil {
		ctx.AbortWithStatusJSON(500, err)
	}
	servicesByName := make(map[string]struct{})
	var servicesNodesCount int
	for _, s := range services {
		if _, ok := servicesByName[s.Name]; !ok {
			servicesByName[s.Name] = struct{}{}
		}
		servicesNodesCount += len(s.Nodes)
	}
	var resp = getSummaryResponse{
		Registry: registrySummary{
			Type:  s.registry.String(),
			Addrs: s.registry.Options().Addrs,
		},
		Services: servicesSummary{
			Count:      len(servicesByName),
			NodesCount: servicesNodesCount,
		},
	}
	ctx.JSON(200, resp)
}

// @ID getVersion
// @Success 200 	{object}	object
// @Router /version [get]
func (s *service) GetVersion(ctx *gin.Context) {
	ctx.JSON(200, gin.H{"version": config.Version})
}
