package routing

import (
	"go-micro.dev/v4/api/client"
)

type Routing interface {
	Directions(*DirectionsRequest) (*DirectionsResponse, error)
	Eta(*EtaRequest) (*EtaResponse, error)
	Route(*RouteRequest) (*RouteResponse, error)
}

func NewRoutingService(token string) *RoutingService {
	return &RoutingService{
		client: client.NewClient(&client.Options{
			Token: token,
		}),
	}
}

type RoutingService struct {
	client *client.Client
}

// Turn by turn directions from a start point to an end point including maneuvers and bearings
func (t *RoutingService) Directions(request *DirectionsRequest) (*DirectionsResponse, error) {

	rsp := &DirectionsResponse{}
	return rsp, t.client.Call("routing", "Directions", request, rsp)

}

// Get the eta for a route from origin to destination. The eta is an estimated time based on car routes
func (t *RoutingService) Eta(request *EtaRequest) (*EtaResponse, error) {

	rsp := &EtaResponse{}
	return rsp, t.client.Call("routing", "Eta", request, rsp)

}

// Retrieve a route as a simple list of gps points along with total distance and estimated duration
func (t *RoutingService) Route(request *RouteRequest) (*RouteResponse, error) {

	rsp := &RouteResponse{}
	return rsp, t.client.Call("routing", "Route", request, rsp)

}

type Direction struct {
	// distance to travel in meters
	Distance float64 `json:"distance"`
	// duration to travel in seconds
	Duration float64 `json:"duration"`
	// human readable instruction
	Instruction string `json:"instruction"`
	// intersections on route
	Intersections []Intersection `json:"intersections"`
	// maneuver to take
	Maneuver *Maneuver `json:"maneuver"`
	// street name or location
	Name string `json:"name"`
	// alternative reference
	Reference string `json:"reference"`
}

type DirectionsRequest struct {
	// The destination of the journey
	Destination *Point `json:"destination"`
	// The staring point for the journey
	Origin *Point `json:"origin"`
}

type DirectionsResponse struct {
	// Turn by turn directions
	Directions []Direction `json:"directions"`
	// Estimated distance of the route in meters
	Distance float64 `json:"distance"`
	// Estimated duration of the route in seconds
	Duration float64 `json:"duration"`
	// The waypoints on the route
	Waypoints []Waypoint `json:"waypoints"`
}

type EtaRequest struct {
	// The end point for the eta calculation
	Destination *Point `json:"destination"`
	// The starting point for the eta calculation
	Origin *Point `json:"origin"`
	// speed in kilometers
	Speed float64 `json:"speed"`
	// type of transport. Only "car" is supported currently.
	Type string `json:"type"`
}

type EtaResponse struct {
	// eta in seconds
	Duration float64 `json:"duration"`
}

type Intersection struct {
	Bearings []float64 `json:"bearings"`
	Location *Point    `json:"location"`
}

type Maneuver struct {
	Action        string  `json:"action"`
	BearingAfter  float64 `json:"bearing_after"`
	BearingBefore float64 `json:"bearing_before"`
	Direction     string  `json:"direction"`
	Location      *Point  `json:"location"`
}

type Point struct {
	// Lat e.g 52.523219
	Latitude float64 `json:"latitude"`
	// Long e.g 13.428555
	Longitude float64 `json:"longitude"`
}

type RouteRequest struct {
	// Point of destination for the trip
	Destination *Point `json:"destination"`
	// Point of origin for the trip
	Origin *Point `json:"origin"`
}

type RouteResponse struct {
	// estimated distance in meters
	Distance float64 `json:"distance"`
	// estimated duration in seconds
	Duration float64 `json:"duration"`
	// waypoints on the route
	Waypoints []Waypoint `json:"waypoints"`
}

type Waypoint struct {
	// gps point coordinates
	Location *Point `json:"location"`
	// street name or related reference
	Name string `json:"name"`
}
