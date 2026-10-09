// Package catalog reads provider model catalogs without sharing credentials
// across origins or keeping a hard-coded model list.
package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// List handles the OpenAI, Anthropic and Gemini catalog response formats.
func List(ctx context.Context, endpoint, key, format string) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	var names []string
	seen := make(map[string]bool)
	next := ""
	for page := 0; page < 100; page++ {
		u, err := url.Parse(endpoint)
		if err != nil {
			return nil, err
		}
		q := u.Query()
		if format == "anthropic" {
			q.Set("limit", "1000")
			if next != "" {
				q.Set("after_id", next)
			}
		}
		if format == "gemini" {
			q.Set("pageSize", "1000")
			if next != "" {
				q.Set("pageToken", next)
			}
		}
		u.RawQuery = q.Encode()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return nil, err
		}
		switch format {
		case "anthropic":
			req.Header.Set("x-api-key", key)
			req.Header.Set("anthropic-version", "2023-06-01")
		case "gemini":
			req.Header.Set("x-goog-api-key", key)
		default:
			if key != "" {
				req.Header.Set("Authorization", "Bearer "+key)
			}
		}
		response, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("model catalog request failed: %w", err)
		}
		if response.StatusCode != http.StatusOK {
			response.Body.Close()
			return nil, fmt.Errorf("model catalog returned HTTP %d", response.StatusCode)
		}
		var data struct {
			Data []struct {
				ID string `json:"id"`
			} `json:"data"`
			Models []struct {
				Name    string   `json:"name"`
				Methods []string `json:"supportedGenerationMethods"`
			} `json:"models"`
			HasMore bool   `json:"has_more"`
			LastID  string `json:"last_id"`
			Next    string `json:"nextPageToken"`
		}
		if format == "together" {
			var entries []struct {
				ID   string `json:"id"`
				Type string `json:"type"`
			}
			err = json.NewDecoder(io.LimitReader(response.Body, 4*1024*1024)).Decode(&entries)
			response.Body.Close()
			if err != nil {
				return nil, fmt.Errorf("invalid model catalog: %w", err)
			}
			for _, entry := range entries {
				if entry.Type == "chat" || entry.Type == "language" || entry.Type == "code" {
					names = append(names, entry.ID)
				}
			}
			return names, nil
		}
		err = json.NewDecoder(io.LimitReader(response.Body, 4*1024*1024)).Decode(&data)
		response.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("invalid model catalog: %w", err)
		}
		for _, entry := range data.Data {
			names = append(names, entry.ID)
		}
		for _, entry := range data.Models {
			for _, method := range entry.Methods {
				if method == "generateContent" {
					names = append(names, strings.TrimPrefix(entry.Name, "models/"))
					break
				}
			}
		}
		next = ""
		if format == "anthropic" && data.HasMore {
			next = data.LastID
			if next == "" {
				return nil, fmt.Errorf("model catalog missing pagination cursor")
			}
		}
		if format == "gemini" {
			next = data.Next
		}
		if next == "" {
			return names, nil
		}
		if seen[next] {
			return nil, fmt.Errorf("model catalog repeated pagination cursor")
		}
		seen[next] = true
	}
	return nil, fmt.Errorf("model catalog exceeds 100 pages")
}
