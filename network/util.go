package network

import (
	pbNet "github.com/asim/go-micro/v3/network/proto"
	"github.com/asim/go-micro/v3/network/router"
)

// routeToProto encodes route into protobuf and returns it
func routeToProto(route router.Route) *pbNet.Route {
	return &pbNet.Route{
		Service: route.Service,
		Address: route.Address,
		Gateway: route.Gateway,
		Network: route.Network,
		Router:  route.Router,
		Link:    route.Link,
		Metric:  int64(route.Metric),
	}
}

// protoToRoute decodes protobuf route into router route and returns it
func protoToRoute(route *pbNet.Route) router.Route {
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
