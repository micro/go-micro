package web

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Brave returns a search adapter for a host-supplied Brave Search API key.
// Credentials are never sent to redirects or fetched result pages.
func Brave(key string) SearchFunc {
	return func(ctx context.Context, query string) ([]Result, error) {
		if key == "" {
			return nil, errors.New("Brave Search requires an API key")
		}
		ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.search.brave.com/res/v1/web/search?q="+url.QueryEscape(query)+"&count=10", nil)
		if err != nil {
			return nil, err
		}
		request.Header.Set("X-Subscription-Token", key)
		client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
		response, err := client.Do(request)
		if err != nil {
			return nil, err
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("Brave Search returned HTTP %d", response.StatusCode)
		}
		var result struct {
			Web struct {
				Results []Result `json:"results"`
			} `json:"web"`
		}
		if err := json.NewDecoder(io.LimitReader(response.Body, 1024*1024)).Decode(&result); err != nil {
			return nil, err
		}
		return result.Web.Results, nil
	}
}
