package holidays

import (
	"github.com/m3o/m3o-go/client"
)

func NewHolidaysService(token string) *HolidaysService {
	return &HolidaysService{
		client: client.NewClient(&client.Options{
			Token: token,
		}),
	}
}

type HolidaysService struct {
	client *client.Client
}

// Get the list of countries that are supported by this API
func (t *HolidaysService) Countries(request *CountriesRequest) (*CountriesResponse, error) {
	rsp := &CountriesResponse{}
	return rsp, t.client.Call("holidays", "Countries", request, rsp)
}

// List the holiday dates for a given country and year
func (t *HolidaysService) List(request *ListRequest) (*ListResponse, error) {
	rsp := &ListResponse{}
	return rsp, t.client.Call("holidays", "List", request, rsp)
}

type CountriesRequest struct {
}

type CountriesResponse struct {
	Countries []Country `json:"countries"`
}

type Country struct {
	// The 2 letter country code (as defined in ISO 3166-1 alpha-2)
	Code string `json:"code"`
	// The English name of the country
	Name string `json:"name"`
}

type Holiday struct {
	// the country this holiday occurs in
	CountryCode string `json:"countryCode"`
	// date of the holiday in yyyy-mm-dd format
	Date string `json:"date"`
	// the local name of the holiday
	LocalName string `json:"localName"`
	// the name of the holiday in English
	Name string `json:"name"`
	// the regions within the country that observe this holiday (if not all of them)
	Regions []string `json:"regions"`
	// the type of holiday Public, Bank, School, Authorities, Optional, Observance
	Types []string `json:"types"`
}

type ListRequest struct {
	// The 2 letter country code (as defined in ISO 3166-1 alpha-2)
	CountryCode string `json:"countryCode"`
	// The year to list holidays for
	Year int64 `json:"year,string"`
}

type ListResponse struct {
	Holidays []Holiday `json:"holidays"`
}
