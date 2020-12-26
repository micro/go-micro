package main

import (
	"encoding/json"
	"log"

	"github.com/micro/go-micro/examples/booking/data"
	"github.com/micro/go-micro/examples/booking/srv/profile/proto"

	"context"
	"golang.org/x/net/trace"

	"github.com/micro/go-micro/v2"
	"github.com/micro/go-micro/v2/metadata"
)

type Profile struct {
	hotels map[string]*profile.Hotel
}

// GetProfiles returns hotel profiles for requested IDs
func (s *Profile) GetProfiles(ctx context.Context, req *profile.Request, rsp *profile.Result) error {
	md, _ := metadata.FromContext(ctx)
	traceID := md["traceID"]
	if tr, ok := trace.FromContext(ctx); ok {
		tr.LazyPrintf("traceID %s", traceID)
	}

	for _, i := range req.HotelIds {
		rsp.Hotels = append(rsp.Hotels, s.hotels[i])
	}
	return nil
}

// loadProfiles loads hotel profiles from a JSON file.
func loadProfiles(path string) map[string]*profile.Hotel {
	file := data.MustAsset(path)

	// unmarshal json profiles
	hotels := []*profile.Hotel{}
	if err := json.Unmarshal(file, &hotels); err != nil {
		log.Fatalf("Failed to load json: %v", err)
	}

	profiles := make(map[string]*profile.Hotel)
	for _, hotel := range hotels {
		profiles[hotel.Id] = hotel
	}
	return profiles
}

func main() {
	service := micro.NewService(
		micro.Name("go.micro.srv.profile"),
	)

	service.Init()

	profile.RegisterProfileHandler(service.Server(), &Profile{
		hotels: loadProfiles("data/profiles.json"),
	})

	service.Run()
}
