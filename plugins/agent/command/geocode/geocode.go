package geocode

/*
	Geocode command for the Micro Bot

	usage: geocode [address]
*/

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/micro/go-micro/v2/agent/command"
)

func init() {
	command.Commands["^geocode "] = Geocode()
}

func Geocode() command.Command {
	usage := "geocode [address]"
	desc := "Returns the geocoded address; lat,lng"

	type Location struct {
		Lat float64 `json:"lat"`
		Lng float64 `json:"lng"`
	}

	type Geometry struct {
		Location Location `json:"location"`
	}

	type Result struct {
		Geometry Geometry `json:"geometry"`
	}

	type Results struct {
		Results []Result `json:"results"`
		Status  string   `json:"status"`
	}

	return command.NewCommand("geocode", usage, desc, func(args ...string) ([]byte, error) {
		if len(args) < 2 {
			return []byte("geocode what?"), nil
		}
		u := url.Values{}
		u.Set("address", strings.Join(args[1:], " "))

		rsp, err := http.Get("https://maps.googleapis.com/maps/api/geocode/json?" + u.Encode())
		if err != nil {
			return nil, err
		}
		defer rsp.Body.Close()

		var res Results
		if err := json.NewDecoder(rsp.Body).Decode(&res); err != nil {
			return nil, err
		}

		if res.Status != "OK" {
			return nil, fmt.Errorf("returned status: %s", res.Status)
		}

		lat := res.Results[0].Geometry.Location.Lat
		lng := res.Results[0].Geometry.Location.Lng
		val := fmt.Sprintf("%.6f,%.6f", lat, lng)

		return []byte(val), nil
	})
}
