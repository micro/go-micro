package carbon

import (
	"go-micro.dev/v4/api/client"
)

type Carbon interface {
	Offset(*OffsetRequest) (*OffsetResponse, error)
}

func NewCarbonService(token string) *CarbonService {
	return &CarbonService{
		client: client.NewClient(&client.Options{
			Token: token,
		}),
	}
}

type CarbonService struct {
	client *client.Client
}

// Purchase 1KG (0.001 tonne) of carbon offsets in a single request
func (t *CarbonService) Offset(request *OffsetRequest) (*OffsetResponse, error) {

	rsp := &OffsetResponse{}
	return rsp, t.client.Call("carbon", "Offset", request, rsp)

}

type OffsetRequest struct {
}

type OffsetResponse struct {
	// the metric used e.g KG or Tonnes
	Metric string `json:"metric"`
	// projects it was allocated to
	Projects []Project `json:"projects"`
	// number of tonnes
	Tonnes float64 `json:"tonnes"`
	// number of units purchased
	Units int32 `json:"units"`
}

type Project struct {
	// name of the project
	Name string `json:"name"`
	// percentage that went to this
	Percentage float64 `json:"percentage"`
	// amount in tonnes
	Tonnes float64 `json:"tonnes"`
}
