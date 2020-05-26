// Package proto contains utility functions for working with protobufs
package proto

import (
	"github.com/micro/go-micro/v2/router"
	pbRtr "github.com/micro/go-micro/v2/router/service/proto"
)

// RouteToProto encodes route into protobuf and returns it
func RouteToProto(route router.Route) *pbRtr.Route {
	return &pbRtr.Route{
		Service: route.Service,
		Address: route.Address,
		Gateway: route.Gateway,
		Network: route.Network,
		Router:  route.Router,
		Link:    route.Link,
		Metric:  int64(route.Metric),
	}
}

// ProtoToRoute decodes protobuf route into router route and returns it
func ProtoToRoute(route *pbRtr.Route) router.Route {
	return router.Route{
		Service: route.Service,
		Address: route.Address,
		Gateway: route.Gateway,
		Network: route.Network,
		Router:  route.Router,
		Link:    route.Link,
		Metric:  route.Metric,
	}
}
