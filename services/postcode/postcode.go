package postcode

import (
	"github.com/m3o/m3o-go/client"
)

func NewPostcodeService(token string) *PostcodeService {
	return &PostcodeService{
		client: client.NewClient(&client.Options{
			Token: token,
		}),
	}
}

type PostcodeService struct {
	client *client.Client
}

// Lookup a postcode to retrieve the related region, county, etc
func (t *PostcodeService) Lookup(request *LookupRequest) (*LookupResponse, error) {
	rsp := &LookupResponse{}
	return rsp, t.client.Call("postcode", "Lookup", request, rsp)
}

// Return a random postcode and its related info
func (t *PostcodeService) Random(request *RandomRequest) (*RandomResponse, error) {
	rsp := &RandomResponse{}
	return rsp, t.client.Call("postcode", "Random", request, rsp)
}

// Validate a postcode.
func (t *PostcodeService) Validate(request *ValidateRequest) (*ValidateResponse, error) {
	rsp := &ValidateResponse{}
	return rsp, t.client.Call("postcode", "Validate", request, rsp)
}

type LookupRequest struct {
	// UK postcode e.g SW1A 2AA
	Postcode string `json:"postcode"`
}

type LookupResponse struct {
	// country e.g United Kingdom
	Country string `json:"country"`
	// e.g Westminster
	District string `json:"district"`
	// e.g 51.50354
	Latitude float64 `json:"latitude"`
	// e.g -0.127695
	Longitude float64 `json:"longitude"`
	// UK postcode e.g SW1A 2AA
	Postcode string `json:"postcode"`
	// related region e.g London
	Region string `json:"region"`
	// e.g St James's
	Ward string `json:"ward"`
}

type RandomRequest struct {
}

type RandomResponse struct {
	// country e.g United Kingdom
	Country string `json:"country"`
	// e.g Westminster
	District string `json:"district"`
	// e.g 51.50354
	Latitude float64 `json:"latitude"`
	// e.g -0.127695
	Longitude float64 `json:"longitude"`
	// UK postcode e.g SW1A 2AA
	Postcode string `json:"postcode"`
	// related region e.g London
	Region string `json:"region"`
	// e.g St James's
	Ward string `json:"ward"`
}

type ValidateRequest struct {
	// UK postcode e.g SW1A 2AA
	Postcode string `json:"postcode"`
}

type ValidateResponse struct {
	// Is the postcode valid (true) or not (false)
	Valid bool `json:"valid"`
}
