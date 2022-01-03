package vehicle

import (
	"go-micro.dev/v4/api/client"
)

type Vehicle interface {
	Lookup(*LookupRequest) (*LookupResponse, error)
}

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
	Co2Emissions float64 `json:"co2_emissions"`
	// colour of vehicle
	Colour string `json:"colour"`
	// engine capacity
	EngineCapacity int32 `json:"engine_capacity"`
	// fuel type e.g petrol, diesel
	FuelType string `json:"fuel_type"`
	// date of last v5 issue
	LastV5Issued string `json:"last_v5_issued"`
	// make of vehicle
	Make string `json:"make"`
	// month of first registration
	MonthOfFirstRegistration string `json:"month_of_first_registration"`
	// mot expiry
	MotExpiry string `json:"mot_expiry"`
	// mot status
	MotStatus string `json:"mot_status"`
	// registration number
	Registration string `json:"registration"`
	// tax due data
	TaxDueDate string `json:"tax_due_date"`
	// tax status
	TaxStatus string `json:"tax_status"`
	// type approvale
	TypeApproval string `json:"type_approval"`
	// wheel plan
	Wheelplan string `json:"wheelplan"`
	// year of manufacture
	YearOfManufacture int32 `json:"year_of_manufacture"`
}
