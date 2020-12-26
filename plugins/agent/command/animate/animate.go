package animate

/*
	Animate command for the Micro Bot

	usage: animate [text]
*/

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/micro/go-micro/v2/agent/command"
)

var (
	// public giphy APIKey
	// Deprecated
	APIKey = "dc6zaTOxFJmzC"
)

func init() {
	command.Commands["^animate "] = Animate()
}

func Animate() command.Command {
	usage := "animate [text]"
	desc := "Returns an animation"

	type Meta struct {
		Status int    `json:"status"`
		Msg    string `json:"msg"`
	}

	type Image struct {
		Url string `json:"url"`
	}

	type Images struct {
		FixedHeight Image `json:"fixed_height"`
	}

	type Result struct {
		Images Images `json:"images"`
	}

	type Results struct {
		Data []Result `json:"data"`
		Meta Meta     `json:"meta"`
	}

	return command.NewCommand("animate", usage, desc, func(args ...string) ([]byte, error) {
		if len(args) < 2 {
			return []byte("animate what?"), nil
		}
		u := url.Values{}
		u.Set("q", strings.Join(args[1:], " "))
		u.Set("limit", "1")
		u.Set("api_APIKey", APIKey)

		rsp, err := http.Get("http://api.giphy.com/v1/gifs/search?" + u.Encode())
		if err != nil {
			return nil, err
		}
		defer rsp.Body.Close()

		var res Results
		if err := json.NewDecoder(rsp.Body).Decode(&res); err != nil {
			return nil, err
		}

		if res.Meta.Status != 200 {
			return nil, fmt.Errorf("returned status: %d %s", res.Meta.Status, res.Meta.Msg)
		}

		if len(res.Data) == 0 {
			return nil, nil
		}

		return []byte(res.Data[0].Images.FixedHeight.Url), nil
	})
}
