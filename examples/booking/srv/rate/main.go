package main

import (
	"encoding/json"
	"log"

	"github.com/micro/go-micro/examples/booking/data"
	"github.com/micro/go-micro/examples/booking/srv/rate/proto"

	"context"
	"golang.org/x/net/trace"

	"github.com/micro/go-micro/v2"
	"github.com/micro/go-micro/v2/metadata"
)

type stay struct {
	HotelID string
	InDate  string
	OutDate string
}

type Rate struct {
	rateTable map[stay]*rate.RatePlan
}

// GetRates gets rates for hotels for specific date range.
func (s *Rate) GetRates(ctx context.Context, req *rate.Request, rsp *rate.Result) error {
	md, _ := metadata.FromContext(ctx)
	traceID := md["traceID"]

	if tr, ok := trace.FromContext(ctx); ok {
		tr.LazyPrintf("traceID %s", traceID)
	}

	for _, hotelID := range req.HotelIds {
		stay := stay{
			HotelID: hotelID,
			InDate:  req.InDate,
			OutDate: req.OutDate,
		}
		if s.rateTable[stay] != nil {
			rsp.RatePlans = append(rsp.RatePlans, s.rateTable[stay])
		}
	}
	return nil
}

// loadRates loads rate codes from JSON file.
func loadRateTable(path string) map[stay]*rate.RatePlan {
	file := data.MustAsset("data/rates.json")

	rates := []*rate.RatePlan{}
	if err := json.Unmarshal(file, &rates); err != nil {
		log.Fatalf("Failed to load json: %v", err)
	}

	rateTable := make(map[stay]*rate.RatePlan)
	for _, ratePlan := range rates {
		stay := stay{
			HotelID: ratePlan.HotelId,
			InDate:  ratePlan.InDate,
			OutDate: ratePlan.OutDate,
		}
		rateTable[stay] = ratePlan
	}
	return rateTable
}

func main() {
	service := micro.NewService(
		micro.Name("go.micro.srv.rate"),
	)

	service.Init()

	rate.RegisterRateHandler(service.Server(), &Rate{
		rateTable: loadRateTable("data/rates.json"),
	})

	service.Run()
}
