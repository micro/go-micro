package evchargers

import (
	"go-micro.dev/v4/api/client"
)

type Evchargers interface {
	ReferenceData(*ReferenceDataRequest) (*ReferenceDataResponse, error)
	Search(*SearchRequest) (*SearchResponse, error)
}

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
	AccessComments  string       `json:"access_comments"`
	AddressLine1    string       `json:"address_line_1"`
	AddressLine2    string       `json:"address_line_2"`
	Country         *Country     `json:"country"`
	CountryId       string       `json:"country_id"`
	LatLng          string       `json:"lat_lng"`
	Location        *Coordinates `json:"location"`
	Postcode        string       `json:"postcode"`
	StateOrProvince string       `json:"state_or_province"`
	Title           string       `json:"title"`
	Town            string       `json:"town"`
}

type BoundingBox struct {
	BottomLeft *Coordinates `json:"bottom_left"`
	TopRight   *Coordinates `json:"top_right"`
}

type ChargerType struct {
	Comments string `json:"comments"`
	Id       string `json:"id"`
	// Is this 40KW+
	IsFastChargeCapable bool   `json:"is_fast_charge_capable"`
	Title               string `json:"title"`
}

type CheckinStatusType struct {
	Id          string `json:"id"`
	IsAutomated bool   `json:"is_automated"`
	IsPositive  bool   `json:"is_positive"`
	Title       string `json:"title"`
}

type Connection struct {
	// The amps offered
	Amps           float64         `json:"amps"`
	ConnectionType *ConnectionType `json:"connection_type"`
	// The ID of the connection type
	ConnectionTypeId string `json:"connection_type_id"`
	// The current
	Current string       `json:"current"`
	Level   *ChargerType `json:"level"`
	// The level of charging power available
	LevelId string `json:"level_id"`
	// The power in KW
	Power     float64 `json:"power"`
	Reference string  `json:"reference"`
	// The voltage offered
	Voltage float64 `json:"voltage"`
}

type ConnectionType struct {
	FormalName     string `json:"formal_name"`
	Id             string `json:"id"`
	IsDiscontinued bool   `json:"is_discontinued"`
	IsObsolete     bool   `json:"is_obsolete"`
	Title          string `json:"title"`
}

type Coordinates struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type Country struct {
	ContinentCode string `json:"continent_code"`
	Id            string `json:"id"`
	IsoCode       string `json:"iso_code"`
	Title         string `json:"title"`
}

type CurrentType struct {
	Description string `json:"description"`
	Id          string `json:"id"`
	Title       string `json:"title"`
}

type DataProvider struct {
	Comments               string                  `json:"comments"`
	DataProviderStatusType *DataProviderStatusType `json:"data_provider_status_type"`
	Id                     string                  `json:"id"`
	// How is this data licensed
	License string `json:"license"`
	Title   string `json:"title"`
	Website string `json:"website"`
}

type DataProviderStatusType struct {
	Id                string `json:"id"`
	IsProviderEnabled bool   `json:"is_provider_enabled"`
	Title             string `json:"title"`
}

type Operator struct {
	Comments         string `json:"comments"`
	ContactEmail     string `json:"contact_email"`
	FaultReportEmail string `json:"fault_report_email"`
	Id               string `json:"id"`
	// Is this operator a private individual vs a company
	IsPrivateIndividual bool   `json:"is_private_individual"`
	PhonePrimary        string `json:"phone_primary"`
	PhoneSecondary      string `json:"phone_secondary"`
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
	DataProviderId string `json:"data_provider_id"`
	// The ID of the charger
	Id string `json:"id"`
	// The number of charging points
	NumPoints int64 `json:"num_points,string"`
	// The operator
	Operator *Operator `json:"operator"`
	// The ID of the operator of the charger
	OperatorId string `json:"operator_id"`
	// The type of usage
	UsageType *UsageType `json:"usage_type"`
	// The type of usage for this charger point (is it public, membership required, etc)
	UsageTypeId string `json:"usage_type_id"`
}

type ReferenceDataRequest struct {
}

type ReferenceDataResponse struct {
	// The types of charger
	ChargerTypes *ChargerType `json:"charger_types"`
	// The types of checkin status
	CheckinStatusTypes *CheckinStatusType `json:"checkin_status_types"`
	// The types of connection
	ConnectionTypes *ConnectionType `json:"connection_types"`
	// The countries
	Countries []Country `json:"countries"`
	// The types of current
	CurrentTypes *CurrentType `json:"current_types"`
	// The providers of the charger data
	DataProviders *DataProvider `json:"data_providers"`
	// The companies operating the chargers
	Operators []Operator `json:"operators"`
	// The status of the charger
	StatusTypes *StatusType `json:"status_types"`
	// The status of a submission
	SubmissionStatusTypes *SubmissionStatusType `json:"submission_status_types"`
	// The different types of usage
	UsageTypes *UsageType `json:"usage_types"`
	// The types of user comment
	UserCommentTypes *UserCommentType `json:"user_comment_types"`
}

type SearchRequest struct {
	// Bounding box to search within (top left and bottom right coordinates)
	Box *BoundingBox `json:"box"`
	// IDs of the connection type
	ConnectionTypes string `json:"connection_types"`
	// Country ID
	CountryId string `json:"country_id"`
	// Search distance from point in metres, defaults to 5000m
	Distance int64 `json:"distance,string"`
	// Supported charging levels
	Levels []string `json:"levels"`
	// Coordinates from which to begin search
	Location *Coordinates `json:"location"`
	// Maximum number of results to return, defaults to 100
	MaxResults int64 `json:"max_results,string"`
	// Minimum power in KW. Note: data not available for many chargers
	MinPower int64 `json:"min_power,string"`
	// IDs of the the EV charger operator
	Operators []string `json:"operators"`
	// Usage of the charge point (is it public, membership required, etc)
	UsageTypes string `json:"usage_types"`
}

type SearchResponse struct {
	Pois []Poi `json:"pois"`
}

type StatusType struct {
	Id            string `json:"id"`
	IsOperational bool   `json:"is_operational"`
	Title         string `json:"title"`
}

type SubmissionStatusType struct {
	Id     string `json:"id"`
	IsLive bool   `json:"is_live"`
	Title  string `json:"title"`
}

type UsageType struct {
	Id                   string `json:"id"`
	IsAccessKeyRequired  bool   `json:"is_access_key_required"`
	IsMembershipRequired bool   `json:"is_membership_required"`
	IsPayAtLocation      bool   `json:"is_pay_at_location"`
	Title                string `json:"title"`
}

type UserCommentType struct {
	Id    string `json:"id"`
	Title string `json:"title"`
}
