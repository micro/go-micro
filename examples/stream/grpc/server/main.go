package main

import (
	"context"
	"io"
	"sync"
	"time"

	pb "github.com/asim/go-micro/examples/v4/stream/grpc/proto"
	"github.com/asim/go-micro/plugins/server/grpc/v4"
	"go-micro.dev/v4"
	"go-micro.dev/v4/logger"
	"google.golang.org/protobuf/proto"
)

type server struct {
	mu         sync.Mutex
	routeNotes map[string][]*pb.RouteNote
}

func main() {
	srv := micro.NewService(
		micro.Server(grpc.NewServer()),
		micro.Name("stream-server"),
	)
	srv.Init()
	pb.RegisterRouteGuideHandler(srv.Server(), &server{routeNotes: make(map[string][]*pb.RouteNote)})
	if err := srv.Run(); err != nil {
		logger.Fatal(err)
	}
}

func (s *server) GetFeature(ctx context.Context, in *pb.Point, out *pb.Feature) error {
	for _, f := range features {
		if proto.Equal(f.Location, in) {
			out.Location = f.Location
			out.Name = f.Name
			return nil
		}
	}
	out.Location = in
	return nil
}

func (s *server) ListFeatures(ctx context.Context, in *pb.Rectangle, stream pb.RouteGuide_ListFeaturesStream) error {
	for _, feature := range features {
		if inRange(feature.Location, in) {
			if err := stream.Send(feature); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *server) RecordRoute(ctx context.Context, stream pb.RouteGuide_RecordRouteStream) error {
	var pointCount, featureCount, distance int32
	var lastPoint *pb.Point
	startTime := time.Now()
	for {
		point, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		pointCount++
		for _, feature := range features {
			if proto.Equal(feature.Location, point) {
				featureCount++
			}
		}
		if lastPoint != nil {
			distance += calcDistance(lastPoint, point)
		}
		lastPoint = point
	}
	return stream.SendMsg(&pb.RouteSummary{
		PointCount:   pointCount,
		FeatureCount: featureCount,
		Distance:     distance,
		ElapsedTime:  int32(time.Since(startTime).Seconds()),
	})
}

func (s *server) RouteChat(ctx context.Context, stream pb.RouteGuide_RouteChatStream) error {
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		key := serialize(in.Location)

		s.mu.Lock()
		s.routeNotes[key] = append(s.routeNotes[key], in)
		// Note: this copy prevents blocking other clients while serving this one.
		// We don't need to do a deep copy, because elements in the slice are
		// insert-only and never modified.
		rn := make([]*pb.RouteNote, len(s.routeNotes[key]))
		copy(rn, s.routeNotes[key])
		s.mu.Unlock()

		for _, note := range rn {
			if err := stream.Send(note); err != nil {
				return err
			}
		}
	}
	return nil
}
