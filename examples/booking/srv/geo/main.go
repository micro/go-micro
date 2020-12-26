package main

import (
	"encoding/json"
	"log"

	"github.com/hailocab/go-geoindex"
	"github.com/micro/go-micro/examples/booking/data"
	"github.com/micro/go-micro/examples/booking/srv/geo/proto"

	"context"
	"golang.org/x/net/trace"

	"github.com/micro/go-micro/v2"
	"github.com/micro/go-micro/v2/metadata"
)

const (
	maxSearchRadius  = 10
	maxSearchResults = 5
)

type point struct {
	Pid  string  `json:"hotelId"`
	Plat float64 `json:"lat"`
	Plon float64 `json:"lon"`
}

// Implement Point interface
func (p *point) Lat() float64 { return p.Plat }
func (p *point) Lon() float64 { return p.Plon }
func (p *point) Id() string   { return p.Pid }

type Geo struct {
	index *geoindex.ClusteringIndex
}

// Nearby returns all hotels within a given distance.
func (s *Geo) Nearby(ctx context.Context, req *geo.Request, rsp *geo.Result) error {
	md, _ := metadata.FromContext(ctx)
	traceID := md["traceID"]

	if tr, ok := trace.FromContext(ctx); ok {
		tr.LazyPrintf("traceID %s", traceID)
	}

	// create center point for query
	center := &geoindex.GeoPoint{
		Pid:  "",
		Plat: float64(req.Lat),
		Plon: float64(req.Lon),
	}

	// find points around center point
	points := s.index.KNearest(center, maxSearchResults, geoindex.Km(maxSearchRadius), func(p geoindex.Point) bool {
		return true
	})

	for _, p := range points {
		rsp.HotelIds = append(rsp.HotelIds, p.Id())
	}
	return nil
}

// newGeoIndex returns a geo index with points loaded
func newGeoIndex(path string) *geoindex.ClusteringIndex {
	file := data.MustAsset(path)

	// unmarshal json points
	var points []*point
	if err := json.Unmarshal(file, &points); err != nil {
		log.Fatalf("Failed to load hotels: %v", err)
	}

	// add points to index
	index := geoindex.NewClusteringIndex()
	for _, point := range points {
		index.Add(point)
	}
	return index
}

func main() {
	service := micro.NewService(
		micro.Name("go.micro.srv.geo"),
	)

	service.Init()

	geo.RegisterGeoHandler(service.Server(), &Geo{
		index: newGeoIndex("data/locations.json"),
	})

	service.Run()
}
