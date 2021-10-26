package main

import (
	"context"
	"io"
	"math/rand"
	"time"

	pb "github.com/asim/go-micro/examples/v4/stream/grpc/proto"
	"github.com/asim/go-micro/plugins/client/grpc/v4"
	"go-micro.dev/v4"
	"go-micro.dev/v4/logger"
)

func main() {
	srv := micro.NewService(
		micro.Client(grpc.NewClient()),
		micro.Name("stream-client"),
	)
	srv.Init()
	client := pb.NewRouteGuideService("stream-server", srv.Client())

	for {
		// Looking for a valid feature
		printFeature(client, &pb.Point{Latitude: 409146138, Longitude: -746188906})
		// Feature missing.
		printFeature(client, &pb.Point{Latitude: 0, Longitude: 0})
		// Looking for features between 40, -75 and 42, -73.
		printFeatures(client, &pb.Rectangle{
			Lo: &pb.Point{Latitude: 400000000, Longitude: -750000000},
			Hi: &pb.Point{Latitude: 420000000, Longitude: -730000000},
		})

		// RecordRoute
		runRecordRoute(client)

		// RouteChat
		runRouteChat(client)

		time.Sleep(time.Second)
	}
}

// printFeature gets the feature for the given point.
func printFeature(client pb.RouteGuideService, point *pb.Point) {
	logger.Info("Getting feature for point (%d, %d)", point.Latitude, point.Longitude)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	feature, err := client.GetFeature(ctx, point)
	if err != nil {
		logger.Fatal(err)
	}
	logger.Info(feature)
}

// printFeatures lists all the features within the given bounding Rectangle.
func printFeatures(client pb.RouteGuideService, rect *pb.Rectangle) {
	logger.Infof("Looking for features within %v", rect)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	stream, err := client.ListFeatures(ctx, rect)
	if err != nil {
		logger.Fatal(err)
	}
	// IMPORTANT: do not forgot to close stream
	defer stream.Close()
	for {
		feature, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			logger.Fatal(err)
		}
		logger.Infof("Feature: name: %q, point:(%v, %v)", feature.GetName(),
			feature.GetLocation().GetLatitude(), feature.GetLocation().GetLongitude())
	}
}

// runRecordRoute sends a sequence of points to server and expects to get a RouteSummary from server.
func runRecordRoute(client pb.RouteGuideService) {
	// Create a random number of random points
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	pointCount := int(r.Int31n(100)) + 2 // Traverse at least two points
	var points []*pb.Point
	for i := 0; i < pointCount; i++ {
		points = append(points, randomPoint(r))
	}
	logger.Infof("Traversing %d points.", len(points))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	stream, err := client.RecordRoute(ctx)
	if err != nil {
		logger.Fatal(err)
	}
	// IMPORTANT: do not forgot to close stream
	defer stream.Close()
	for _, point := range points {
		if err := stream.Send(point); err != nil {
			logger.Fatal(err)
		}
	}
	if err := stream.CloseSend(); err != nil {
		logger.Fatal(err)
	}
	summary := pb.RouteSummary{}
	if err := stream.RecvMsg(&summary); err != nil {
		logger.Fatal(err)
	}
	logger.Infof("Route summary: %v", &summary)
}

// runRouteChat receives a sequence of route notes, while sending notes for various locations.
func runRouteChat(client pb.RouteGuideService) {
	notes := []*pb.RouteNote{
		{Location: &pb.Point{Latitude: 0, Longitude: 1}, Message: "First message"},
		{Location: &pb.Point{Latitude: 0, Longitude: 2}, Message: "Second message"},
		{Location: &pb.Point{Latitude: 0, Longitude: 3}, Message: "Third message"},
		{Location: &pb.Point{Latitude: 0, Longitude: 1}, Message: "Fourth message"},
		{Location: &pb.Point{Latitude: 0, Longitude: 2}, Message: "Fifth message"},
		{Location: &pb.Point{Latitude: 0, Longitude: 3}, Message: "Sixth message"},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	stream, err := client.RouteChat(ctx)
	if err != nil {
		logger.Fatal(err)
	}
	// IMPORTANT: do not forgot to close stream
	defer stream.Close()
	waitc := make(chan struct{})
	go func() {
		for {
			in, err := stream.Recv()
			if err == io.EOF {
				// read done.
				close(waitc)
				break
			}
			if err != nil {
				logger.Fatal(err)
			}
			logger.Infof("Got message %s at point(%d, %d)", in.Message, in.Location.Latitude, in.Location.Longitude)
		}
	}()
	for _, note := range notes {
		if err := stream.Send(note); err != nil {
			logger.Fatal(err)
		}
	}
	if err := stream.CloseSend(); err != nil {
		logger.Fatal(err)
	}
	<-waitc
}

func randomPoint(r *rand.Rand) *pb.Point {
	lat := (r.Int31n(180) - 90) * 1e7
	long := (r.Int31n(360) - 180) * 1e7
	return &pb.Point{Latitude: lat, Longitude: long}
}
