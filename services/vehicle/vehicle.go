package vehicle

import (
	"github.com/m3o/m3o-go/client"
)

func NewVehicleService(token string) *VehicleService {
	return &VehicleService{
		client: client.NewClient(&client.Options{
			Token: token,
		}),
	}
}

type VehicleService struct {
	client *client.Client
}

// Lookup a UK vehicle by it's registration number
func (t *VehicleService) Lookup(request *LookupRequest) (*LookupResponse, error) {
	rsp := &LookupResponse{}
	return rsp, t.client.Call("vehicle", "Lookup", request, rsp)
}

type LookupRequest struct {
	// the vehicle registration number
	Registration string `json:"registration"`
}

type LookupResponse struct {
	// co2 emmissions
	Co2emissions float64 `json:"co2emissions"`
	// colour of vehicle
	Colour string `json:"colour"`
	// engine capacity
	EngineCapacity int32 `json:"engineCapacity"`
	// fuel type e.g petrol, diesel
	FuelType string `json:"fuelType"`
	// date of last v5 issue
	LastV5issued string `json:"lastV5issued"`
	// make of vehicle
	Make string `json:"make"`
	// month of first registration
	MonthOfFirstRegistration string `json:"monthOfFirstRegistration"`
	// mot expiry
	MotExpiry string `json:"motExpiry"`
	// mot status
	MotStatus string `json:"motStatus"`
	// registration number
	Registration string `json:"registration"`
	// tax due data
	TaxDueDate string `json:"taxDueDate"`
	// tax status
	TaxStatus string `json:"taxStatus"`
	// type approvale
	TypeApproval string `json:"typeApproval"`
	// wheel plan
	Wheelplan string `json:"wheelplan"`
	// year of manufacture
	YearOfManufacture int32 `json:"yearOfManufacture"`
}
