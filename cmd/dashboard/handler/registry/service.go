package registry

import (
	"sort"

	"github.com/asim/go-micro/cmd/dashboard/v4/handler/route"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/render"
	"go-micro.dev/v4/registry"
)

type service struct {
	registry registry.Registry
}

func NewRouteRegistrar(registry registry.Registry) route.Registrar {
	return service{registry: registry}
}

func (s service) RegisterRoute(router gin.IRoutes) {
	router.Use(route.AuthRequired()).
		GET("/api/registry/services", s.GetServices).
		GET("/api/registry/service", s.GetServiceDetail).
		GET("/api/registry/service/handlers", s.GetServiceHandlers).
		GET("/api/registry/service/subscribers", s.GetServiceSubscribers)
}

// @Security ApiKeyAuth
// @Tags Registry
// @ID registry_getServices
// @Success 200 	{object}	getServiceListResponse
// @Failure 400 	{object}	string
// @Failure 401 	{object}	string
// @Failure 500		{object}	string
// @Router /api/registry/services [get]
func (s *service) GetServices(ctx *gin.Context) {
	services, err := s.registry.ListServices()
	if err != nil {
		ctx.Render(500, render.String{Format: err.Error()})
		return
	}
	tmp := make(map[string][]string)
	resp := getServiceListResponse{Services: make([]registryServiceSummary, 0, len(services))}
	for _, s := range services {
		if sr, ok := tmp[s.Name]; ok {
			sr = append(sr, s.Version)
			tmp[s.Name] = sr
		} else {
			tmp[s.Name] = []string{s.Version}
		}
	}
	for k, v := range tmp {
		sort.Strings(v)
		resp.Services = append(resp.Services, registryServiceSummary{Name: k, Versions: v})
	}
	sort.Slice(resp.Services, func(i, j int) bool {
		return resp.Services[i].Name < resp.Services[j].Name
	})
	ctx.JSON(200, resp)
}

// @Security ApiKeyAuth
// @Tags Registry
// @ID registry_getServiceDetail
// @Param 	name  	query 		string 						true "service name"
// @Param 	version	query 		string	 					false "service version"
// @Success 200 	{object}	getServiceDetailResponse
// @Failure 400 	{object}	string
// @Failure 401 	{object}	string
// @Failure 500		{object}	string
// @Router /api/registry/service [get]
func (s *service) GetServiceDetail(ctx *gin.Context) {
	name := ctx.Query("name")
	if len(name) == 0 {
		ctx.Render(400, render.String{Format: "service name required"})
		return
	}
	services, err := s.registry.GetService(name)
	if err != nil {
		ctx.Render(500, render.String{Format: err.Error()})
		return
	}
	version := ctx.Query("version")
	resp := getServiceDetailResponse{Services: make([]registryService, 0, len(services))}
	for _, s := range services {
		if len(version) > 0 && s.Version != version {
			continue
		}
		handlers := make([]registryEndpoint, 0)
		subscribers := make([]registryEndpoint, 0)
		for _, e := range s.Endpoints {
			if isSubscriber(e) {
				subscribers = append(subscribers, registryEndpoint{
					Name:     e.Metadata["topic"],
					Request:  convertRegistryValue(e.Request),
					Metadata: e.Metadata,
				})
			} else {
				handlers = append(handlers, registryEndpoint{
					Name:     e.Name,
					Request:  convertRegistryValue(e.Request),
					Response: convertRegistryValue(e.Response),
					Metadata: e.Metadata,
				})
			}
		}
		nodes := make([]registryNode, 0, len(s.Nodes))
		for _, n := range s.Nodes {
			nodes = append(nodes, registryNode{
				Id:       n.Id,
				Address:  n.Address,
				Metadata: n.Metadata,
			})
		}
		resp.Services = append(resp.Services, registryService{
			Name:        s.Name,
			Version:     s.Version,
			Metadata:    s.Metadata,
			Handlers:    handlers,
			Subscribers: subscribers,
			Nodes:       nodes,
		})
	}
	ctx.JSON(200, resp)
}

// @Security ApiKeyAuth
// @Tags Registry
// @ID registry_getServiceHandlers
// @Param 	name  	query 		string 						true "service name"
// @Param 	version	query 		string	 					false "service version"
// @Success 200 	{object}	getServiceHandlersResponse
// @Failure 400 	{object}	string
// @Failure 401 	{object}	string
// @Failure 500		{object}	string
// @Router /api/registry/service/handlers [get]
func (s *service) GetServiceHandlers(ctx *gin.Context) {
	name := ctx.Query("name")
	if len(name) == 0 {
		ctx.Render(400, render.String{Format: "service name required"})
		return
	}
	services, err := s.registry.GetService(name)
	if err != nil {
		ctx.Render(500, render.String{Format: err.Error()})
		return
	}
	version := ctx.Query("version")
	resp := getServiceHandlersResponse{}
	for _, s := range services {
		if s.Version != version {
			continue
		}
		handlers := make([]registryEndpoint, 0, len(s.Endpoints))
		for _, e := range s.Endpoints {
			if isSubscriber(e) {
				continue
			}
			handlers = append(handlers, registryEndpoint{
				Name:    e.Name,
				Request: convertRegistryValue(e.Request),
			})
		}
		resp.Handlers = handlers
		break
	}
	ctx.JSON(200, resp)
}

// @Security ApiKeyAuth
// @Tags Registry
// @ID registry_getServiceSubscribers
// @Param 	name  	query 		string 						true "service name"
// @Param 	version	query 		string	 					false "service version"
// @Success 200 	{object}	getServiceSubscribersResponse
// @Failure 400 	{object}	string
// @Failure 401 	{object}	string
// @Failure 500		{object}	string
// @Router /api/registry/service/subscribers [get]
func (s *service) GetServiceSubscribers(ctx *gin.Context) {
	name := ctx.Query("name")
	if len(name) == 0 {
		ctx.Render(400, render.String{Format: "service name required"})
		return
	}
	services, err := s.registry.GetService(name)
	if err != nil {
		ctx.Render(500, render.String{Format: err.Error()})
		return
	}
	version := ctx.Query("version")
	resp := getServiceSubscribersResponse{}
	for _, s := range services {
		if s.Version != version {
			continue
		}
		subscribers := make([]registryEndpoint, 0, len(s.Endpoints))
		for _, e := range s.Endpoints {
			if !isSubscriber(e) {
				continue
			}
			subscribers = append(subscribers, registryEndpoint{
				Name:    e.Metadata["topic"],
				Request: convertRegistryValue(e.Request),
			})
		}
		resp.Subscribers = subscribers
		break
	}
	ctx.JSON(200, resp)
}

func isSubscriber(ep *registry.Endpoint) bool {
	if ep == nil || len(ep.Metadata) == 0 {
		return false
	}
	if s, ok := ep.Metadata["subscriber"]; ok && s == "true" {
		return true
	}
	return false
}
