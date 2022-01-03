package ip

import (
	"go-micro.dev/v4/api/client"
)

type Ip interface {
	Lookup(*LookupRequest) (*LookupResponse, error)
}

func NewIpService(token string) *IpService {
	return &IpService{
		client: client.NewClient(&client.Options{
			Token: token,
		}),
	}
}

type IpService struct {
	client *client.Client
}

// Lookup the geolocation information for an IP address
func (t *IpService) Lookup(request *LookupRequest) (*LookupResponse, error) {

	rsp := &LookupResponse{}
	return rsp, t.client.Call("ip", "Lookup", request, rsp)

}

type LookupRequest struct {
	// IP to lookup
	Ip string `json:"ip"`
}

type LookupResponse struct {
	// Autonomous system number
	Asn int32 `json:"asn"`
	// Name of the city
	City string `json:"city"`
	// Name of the continent
	Continent string `json:"continent"`
	// Name of the country
	Country string `json:"country"`
	// IP of the query
	Ip string `json:"ip"`
	// Latitude e.g 52.523219
	Latitude float64 `json:"latitude"`
	// Longitude e.g 13.428555
	Longitude float64 `json:"longitude"`
	// Timezone e.g Europe/Rome
	Timezone string `json:"timezone"`
}
