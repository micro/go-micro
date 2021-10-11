package evchargers

import (
	"github.com/m3o/m3o-go/client"
)

func NewEvchargersService(token string) *EvchargersService {
	return &EvchargersService{
		client: client.NewClient(&client.Options{
			Token: token,
		}),
	}
}

type EvchargersService struct {
	client *client.Client
}

// Retrieve reference data as used by this API and in conjunction with the Search endpoint
func (t *EvchargersService) ReferenceData(request *ReferenceDataRequest) (*ReferenceDataResponse, error) {
	rsp := &ReferenceDataResponse{}
	return rsp, t.client.Call("evchargers", "ReferenceData", request, rsp)
}

// Search by giving a coordinate and a max distance, or bounding box and optional filters
func (t *EvchargersService) Search(request *SearchRequest) (*SearchResponse, error) {
	rsp := &SearchResponse{}
	return rsp, t.client.Call("evchargers", "Search", request, rsp)
}

type Address struct {
	// Any comments about how to access the charger
	AccessComments  string       `json:"accessComments"`
	AddressLine1    string       `json:"addressLine1"`
	AddressLine2    string       `json:"addressLine2"`
	Country         *Country     `json:"country"`
	CountryId       string       `json:"countryId"`
	LatLng          string       `json:"latLng"`
	Location        *Coordinates `json:"location"`
	Postcode        string       `json:"postcode"`
	StateOrProvince string       `json:"stateOrProvince"`
	Title           string       `json:"title"`
	Town            string       `json:"town"`
}

type BoundingBox struct {
	BottomLeft *Coordinates `json:"bottomLeft"`
	TopRight   *Coordinates `json:"topRight"`
}

type ChargerType struct {
	Comments string `json:"comments"`
	Id       string `json:"id"`
	// Is this 40KW+
	IsFastChargeCapable bool   `json:"isFastChargeCapable"`
	Title               string `json:"title"`
}

type CheckinStatusType struct {
	Id          string `json:"id"`
	IsAutomated bool   `json:"isAutomated"`
	IsPositive  bool   `json:"isPositive"`
	Title       string `json:"title"`
}

type Connection struct {
	// The amps offered
	Amps           float64         `json:"amps"`
	ConnectionType *ConnectionType `json:"connectionType"`
	// The ID of the connection type
	ConnectionTypeId string `json:"connectionTypeId"`
	// The current
	Current string       `json:"current"`
	Level   *ChargerType `json:"level"`
	// The level of charging power available
	LevelId string `json:"levelId"`
	// The power in KW
	Power     float64 `json:"power"`
	Reference string  `json:"reference"`
	// The voltage offered
	Voltage float64 `json:"voltage"`
}

type ConnectionType struct {
	FormalName     string `json:"formalName"`
	Id             string `json:"id"`
	IsDiscontinued bool   `json:"isDiscontinued"`
	IsObsolete     bool   `json:"isObsolete"`
	Title          string `json:"title"`
}

type Coordinates struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type Country struct {
	ContinentCode string `json:"continentCode"`
	Id            string `json:"id"`
	IsoCode       string `json:"isoCode"`
	Title         string `json:"title"`
}

type CurrentType struct {
	Description string `json:"description"`
	Id          string `json:"id"`
	Title       string `json:"title"`
}

type DataProvider struct {
	Comments               string                  `json:"comments"`
	DataProviderStatusType *DataProviderStatusType `json:"dataProviderStatusType"`
	Id                     string                  `json:"id"`
	// How is this data licensed
	License string `json:"license"`
	Title   string `json:"title"`
	Website string `json:"website"`
}

type DataProviderStatusType struct {
	Id                string `json:"id"`
	IsProviderEnabled bool   `json:"isProviderEnabled"`
	Title             string `json:"title"`
}

type Operator struct {
	Comments         string `json:"comments"`
	ContactEmail     string `json:"contactEmail"`
	FaultReportEmail string `json:"faultReportEmail"`
	Id               string `json:"id"`
	// Is this operator a private individual vs a company
	IsPrivateIndividual bool   `json:"isPrivateIndividual"`
	PhonePrimary        string `json:"phonePrimary"`
	PhoneSecondary      string `json:"phoneSecondary"`
	Title               string `json:"title"`
	Website             string `json:"website"`
}

type Poi struct {
	// The address
	Address *Address `json:"address"`
	// The connections available at this charge point
	Connections []Connection `json:"connections"`
	// The cost of charging
	Cost string `json:"cost"`
	// The ID of the data provider
	DataProviderId string `json:"dataProviderId"`
	// The ID of the charger
	Id string `json:"id"`
	// The number of charging points
	NumPoints int64 `json:"numPoints,string"`
	// The operator
	Operator *Operator `json:"operator"`
	// The ID of the operator of the charger
	OperatorId string `json:"operatorId"`
	// The type of usage
	UsageType *UsageType `json:"usageType"`
	// The type of usage for this charger point (is it public, membership required, etc)
	UsageTypeId string `json:"usageTypeId"`
}

type ReferenceDataRequest struct {
}

type ReferenceDataResponse struct {
	// The types of charger
	ChargerTypes *ChargerType `json:"chargerTypes"`
	// The types of checkin status
	CheckinStatusTypes *CheckinStatusType `json:"checkinStatusTypes"`
	// The types of connection
	ConnectionTypes *ConnectionType `json:"connectionTypes"`
	// The countries
	Countries []Country `json:"countries"`
	// The types of current
	CurrentTypes *CurrentType `json:"currentTypes"`
	// The providers of the charger data
	DataProviders *DataProvider `json:"dataProviders"`
	// The companies operating the chargers
	Operators []Operator `json:"operators"`
	// The status of the charger
	StatusTypes *StatusType `json:"statusTypes"`
	// The status of a submission
	SubmissionStatusTypes *SubmissionStatusType `json:"submissionStatusTypes"`
	// The different types of usage
	UsageTypes *UsageType `json:"usageTypes"`
	// The types of user comment
	UserCommentTypes *UserCommentType `json:"userCommentTypes"`
}

type SearchRequest struct {
	// Bounding box to search within (top left and bottom right coordinates)
	Box *BoundingBox `json:"box"`
	// IDs of the connection type
	ConnectionTypes string `json:"connectionTypes"`
	// Country ID
	CountryId string `json:"countryId"`
	// Search distance from point in metres, defaults to 5000m
	Distance int64 `json:"distance,string"`
	// Supported charging levels
	Levels []string `json:"levels"`
	// Coordinates from which to begin search
	Location *Coordinates `json:"location"`
	// Maximum number of results to return, defaults to 100
	MaxResults int64 `json:"maxResults,string"`
	// Minimum power in KW. Note: data not available for many chargers
	MinPower int64 `json:"minPower,string"`
	// IDs of the the EV charger operator
	Operators []string `json:"operators"`
	// Usage of the charge point (is it public, membership required, etc)
	UsageTypes string `json:"usageTypes"`
}

type SearchResponse struct {
	Pois []Poi `json:"pois"`
}

type StatusType struct {
	Id            string `json:"id"`
	IsOperational bool   `json:"isOperational"`
	Title         string `json:"title"`
}

type SubmissionStatusType struct {
	Id     string `json:"id"`
	IsLive bool   `json:"isLive"`
	Title  string `json:"title"`
}

type UsageType struct {
	Id                   string `json:"id"`
	IsAccessKeyRequired  bool   `json:"isAccessKeyRequired"`
	IsMembershipRequired bool   `json:"isMembershipRequired"`
	IsPayAtLocation      bool   `json:"isPayAtLocation"`
	Title                string `json:"title"`
}

type UserCommentType struct {
	Id    string `json:"id"`
	Title string `json:"title"`
}
