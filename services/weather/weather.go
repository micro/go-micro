package weather

import (
	"github.com/m3o/m3o-go/client"
)

func NewWeatherService(token string) *WeatherService {
	return &WeatherService{
		client: client.NewClient(&client.Options{
			Token: token,
		}),
	}
}

type WeatherService struct {
	client *client.Client
}

// Get the weather forecast for the next 1-10 days
func (t *WeatherService) Forecast(request *ForecastRequest) (*ForecastResponse, error) {
	rsp := &ForecastResponse{}
	return rsp, t.client.Call("weather", "Forecast", request, rsp)
}

// Get the current weather report for a location by postcode, city, zip code, ip address
func (t *WeatherService) Now(request *NowRequest) (*NowResponse, error) {
	rsp := &NowResponse{}
	return rsp, t.client.Call("weather", "Now", request, rsp)
}

type Forecast struct {
	// the average temp in celsius
	AvgTempC float64 `json:"avgTempC"`
	// the average temp in fahrenheit
	AvgTempF float64 `json:"avgTempF"`
	// chance of rain (percentage)
	ChanceOfRain int32 `json:"chanceOfRain"`
	// forecast condition
	Condition string `json:"condition"`
	// date of the forecast
	Date string `json:"date"`
	// the URL of forecast condition icon. Simply prefix with either http or https to use it
	IconUrl string `json:"iconUrl"`
	// max temp in celsius
	MaxTempC float64 `json:"maxTempC"`
	// max temp in fahrenheit
	MaxTempF float64 `json:"maxTempF"`
	// minimum temp in celsius
	MinTempC float64 `json:"minTempC"`
	// minimum temp in fahrenheit
	MinTempF float64 `json:"minTempF"`
	// time of sunrise
	Sunrise string `json:"sunrise"`
	// time of sunset
	Sunset string `json:"sunset"`
	// will it rain
	WillItRain bool `json:"willItRain"`
}

type ForecastRequest struct {
	// number of days. default 1, max 10
	Days int32 `json:"days"`
	// location of the forecase
	Location string `json:"location"`
}

type ForecastResponse struct {
	// country of the request
	Country string `json:"country"`
	// forecast for the next number of days
	Forecast []Forecast `json:"forecast"`
	// e.g 37.55
	Latitude float64 `json:"latitude"`
	// the local time
	LocalTime string `json:"localTime"`
	// location of the request
	Location string `json:"location"`
	// e.g -77.46
	Longitude float64 `json:"longitude"`
	// region related to the location
	Region string `json:"region"`
	// timezone of the location
	Timezone string `json:"timezone"`
}

type NowRequest struct {
	// location to get weather e.g postcode, city
	Location string `json:"location"`
}

type NowResponse struct {
	// cloud cover percentage
	Cloud int32 `json:"cloud"`
	// the weather condition
	Condition string `json:"condition"`
	// country of the request
	Country string `json:"country"`
	// whether its daytime
	Daytime bool `json:"daytime"`
	// feels like in celsius
	FeelsLikeC float64 `json:"feelsLikeC"`
	// feels like in fahrenheit
	FeelsLikeF float64 `json:"feelsLikeF"`
	// the humidity percentage
	Humidity int32 `json:"humidity"`
	// the URL of the related icon. Simply prefix with either http or https to use it
	IconUrl string `json:"iconUrl"`
	// e.g 37.55
	Latitude float64 `json:"latitude"`
	// the local time
	LocalTime string `json:"localTime"`
	// location of the request
	Location string `json:"location"`
	// e.g -77.46
	Longitude float64 `json:"longitude"`
	// region related to the location
	Region string `json:"region"`
	// temperature in celsius
	TempC float64 `json:"tempC"`
	// temperature in fahrenheit
	TempF float64 `json:"tempF"`
	// timezone of the location
	Timezone string `json:"timezone"`
	// wind degree
	WindDegree int32 `json:"windDegree"`
	// wind direction
	WindDirection string `json:"windDirection"`
	// wind in kph
	WindKph float64 `json:"windKph"`
	// wind in mph
	WindMph float64 `json:"windMph"`
}
