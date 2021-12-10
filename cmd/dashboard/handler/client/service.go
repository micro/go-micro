package client

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/asim/go-micro/cmd/dashboard/v4/handler/route"
	cgrpc "github.com/asim/go-micro/plugins/client/grpc/v4"
	chttp "github.com/asim/go-micro/plugins/client/http/v4"
	cmucp "github.com/asim/go-micro/plugins/client/mucp/v4"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/render"
	"go-micro.dev/v4/client"
	"go-micro.dev/v4/errors"
	"go-micro.dev/v4/registry"
	"go-micro.dev/v4/selector"
)

type service struct {
	client   client.Client
	registry registry.Registry

	clients   map[string]client.Client
	clientsMu sync.Mutex
}

func NewRouteRegistrar(client client.Client, registry registry.Registry) route.Registrar {
	return &service{client: client, registry: registry}
}

func (s *service) RegisterRoute(router gin.IRoutes) {
	router.Use(route.AuthRequired()).
		POST("/api/client/call", s.Call).
		POST("/api/client/publish", s.Publish)
}

// @Security ApiKeyAuth
// @Tags Client
// @ID client_call
// @Param	input	body		callRequest		true		"request"
// @Success 200 	{object}	object			"success"
// @Failure 400 	{object}	string
// @Failure 401 	{object}	string
// @Failure 500		{object}	string
// @Router /api/client/call [post]
func (s *service) Call(ctx *gin.Context) {
	var req callRequest
	if err := ctx.ShouldBindJSON(&req); nil != err {
		ctx.Render(400, render.String{Format: err.Error()})
		return
	}
	var callReq json.RawMessage
	if len(req.Request) > 0 {
		if err := json.Unmarshal([]byte(req.Request), &callReq); err != nil {
			ctx.Render(400, render.String{Format: "parse request failed: %s", Data: []interface{}{err.Error()}})
			return
		}
	}
	services, err := s.registry.GetService(req.Service)
	if err != nil {
		ctx.Render(400, render.String{Format: err.Error()})
		return
	}
	var c client.Client
	for _, srv := range services {
		if len(req.Version) > 0 && req.Version != srv.Version {
			continue
		}
		if len(srv.Nodes) == 0 {
			ctx.Render(400, render.String{Format: "service node not found"})
			return
		}
		c = s.getClient(srv.Nodes[0].Metadata["server"])
		break
	}
	if c == nil {
		ctx.Render(400, render.String{Format: "service not found"})
		return
	}
	var resp json.RawMessage
	callOpts := []client.CallOption{}
	if len(req.Version) > 0 {
		callOpts = append(callOpts, client.WithSelectOption(selector.WithFilter(selector.FilterVersion(req.Version))))
	}
	requestOpts := []client.RequestOption{client.WithContentType("application/json")}
	if req.Timeout > 0 {
		callOpts = append(callOpts, client.WithRequestTimeout(time.Duration(req.Timeout)*time.Second))
	}
	if err := c.Call(context.TODO(), client.NewRequest(req.Service, req.Endpoint, callReq, requestOpts...), &resp, callOpts...); err != nil {
		if merr := errors.Parse(err.Error()); merr != nil {
			ctx.JSON(200, gin.H{"success": false, "error": merr})
		} else {
			ctx.JSON(200, gin.H{"success": false, "error": err.Error})
		}
		return
	}
	ctx.JSON(200, resp)
}

// @Security ApiKeyAuth
// @Tags Client
// @ID client_publish
// @Param	input	body		publishRequest	true		"request"
// @Success 200 	{object}	object			"success"
// @Failure 400 	{object}	string
// @Failure 401 	{object}	string
// @Failure 500		{object}	string
// @Router /api/client/publish [post]
func (s *service) Publish(ctx *gin.Context) {
	var req publishRequest
	if err := ctx.ShouldBindJSON(&req); nil != err {
		ctx.Render(400, render.String{Format: err.Error()})
		return
	}
	var msg json.RawMessage
	if len(req.Message) > 0 {
		if err := json.Unmarshal([]byte(req.Message), &msg); err != nil {
			ctx.Render(400, render.String{Format: "parse request failed: %s", Data: []interface{}{err.Error()}})
			return
		}
	}
	err := s.client.Publish(ctx, client.NewMessage(req.Topic, msg, client.WithMessageContentType("application/json")))
	if err != nil {
		if merr := errors.Parse(err.Error()); merr != nil {
			ctx.JSON(200, gin.H{"success": false, "error": merr})
		} else {
			ctx.JSON(200, gin.H{"success": false, "error": err.Error})
		}
		return
	}
	ctx.JSON(200, gin.H{"success": true})
}

func (s *service) getClient(serverType string) client.Client {
	if serverType == s.client.String() {
		return s.client
	}
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()
	if s.clients == nil {
		s.clients = make(map[string]client.Client)
	} else {
		if c, ok := s.clients[serverType]; ok {
			return c
		}
	}
	var c client.Client
	switch serverType {
	case "grpc":
		c = cgrpc.NewClient()
		s.clients[serverType] = c
	case "http":
		c = chttp.NewClient()
		s.clients[serverType] = c
	case "mucp":
		c = cmucp.NewClient()
		s.clients[serverType] = c
	default:
		c = s.client
	}
	return c
}