package geocoding

import (
	"github.com/m3o/m3o-go/client"
)

func NewGeocodingService(token string) *GeocodingService {
	return &GeocodingService{
		client: client.NewClient(&client.Options{
			Token: token,
		}),
	}
}

type GeocodingService struct {
	client *client.Client
}

// Lookup returns a geocoded address including normalized address and gps coordinates. All fields are optional, provide more to get more accurate results
func (t *GeocodingService) Lookup(request *LookupRequest) (*LookupResponse, error) {
	rsp := &LookupResponse{}
	return rsp, t.client.Call("geocoding", "Lookup", request, rsp)
}

// Reverse lookup an address from gps coordinates
func (t *GeocodingService) Reverse(request *ReverseRequest) (*ReverseResponse, error) {
	rsp := &ReverseResponse{}
	return rsp, t.client.Call("geocoding", "Reverse", request, rsp)
}

type Address struct {
	City     string `json:"city"`
	Country  string `json:"country"`
	LineOne  string `json:"lineOne"`
	LineTwo  string `json:"lineTwo"`
	Postcode string `json:"postcode"`
}

type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type LookupRequest struct {
	Address  string `json:"address"`
	City     string `json:"city"`
	Country  string `json:"country"`
	Postcode string `json:"postcode"`
}

type LookupResponse struct {
	Address  *Address  `json:"address"`
	Location *Location `json:"location"`
}

type ReverseRequest struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type ReverseResponse struct {
	Address  *Address  `json:"address"`
	Location *Location `json:"location"`
}
