package handler

import (
	"context"

	"github.com/micro/go-micro/errors"
	"github.com/micro/go-micro/network/router"
	pb "github.com/micro/go-micro/network/router/proto"
)

type Table struct {
	Router router.Router
}

func (t *Table) Create(ctx context.Context, route *pb.Route, resp *pb.CreateResponse) error {
	err := t.Router.Table().Create(router.Route{
		Service: route.Service,
		Address: route.Address,
		Gateway: route.Gateway,
		Network: route.Network,
		Link:    route.Link,
		Metric:  int(route.Metric),
	})
	if err != nil {
		return errors.InternalServerError("go.micro.router", "failed to create route: %s", err)
	}

	return nil
}

func (t *Table) Update(ctx context.Context, route *pb.Route, resp *pb.UpdateResponse) error {
	err := t.Router.Table().Update(router.Route{
		Service: route.Service,
		Address: route.Address,
		Gateway: route.Gateway,
		Network: route.Network,
		Link:    route.Link,
		Metric:  int(route.Metric),
	})
	if err != nil {
		return errors.InternalServerError("go.micro.router", "failed to update route: %s", err)
	}

	return nil
}

func (t *Table) Delete(ctx context.Context, route *pb.Route, resp *pb.DeleteResponse) error {
	err := t.Router.Table().Delete(router.Route{
		Service: route.Service,
		Address: route.Address,
		Gateway: route.Gateway,
		Network: route.Network,
		Link:    route.Link,
		Metric:  int(route.Metric),
	})
	if err != nil {
		return errors.InternalServerError("go.micro.router", "failed to delete route: %s", err)
	}

	return nil
}

// List returns all routes in the routing table
func (t *Table) List(ctx context.Context, req *pb.Request, resp *pb.ListResponse) error {
	routes, err := t.Router.Table().List()
	if err != nil {
		return errors.InternalServerError("go.micro.router", "failed to list routes: %s", err)
	}

	var respRoutes []*pb.Route
	for _, route := range routes {
		respRoute := &pb.Route{
			Service: route.Service,
			Address: route.Address,
			Gateway: route.Gateway,
			Network: route.Network,
			Link:    route.Link,
			Metric:  int64(route.Metric),
		}
		respRoutes = append(respRoutes, respRoute)
	}

	resp.Routes = respRoutes

	return nil
}

func (t *Table) Query(ctx context.Context, req *pb.QueryRequest, resp *pb.QueryResponse) error {
	query := router.NewQuery(
		router.QueryService(req.Query.Service),
	)

	routes, err := t.Router.Table().Query(query)
	if err != nil {
		return errors.InternalServerError("go.micro.router", "failed to lookup routes: %s", err)
	}

	var respRoutes []*pb.Route
	for _, route := range routes {
		respRoute := &pb.Route{
			Service: route.Service,
			Address: route.Address,
			Gateway: route.Gateway,
			Network: route.Network,
			Link:    route.Link,
			Metric:  int64(route.Metric),
		}
		respRoutes = append(respRoutes, respRoute)
	}

	resp.Routes = respRoutes

	return nil
}
