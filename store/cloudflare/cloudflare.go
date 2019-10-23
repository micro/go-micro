// Package cloudflare is a store implementation backed by cloudflare workers kv
// Note that the cloudflare workers KV API is eventually consistent.
package cloudflare

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/micro/go-micro/config/options"
	"github.com/micro/go-micro/store"
	"github.com/pkg/errors"
)

const apiBaseURL = "https://api.cloudflare.com/client/v4/"

type workersKV struct {
	options.Options
	httpClient *http.Client
}

// New returns a cloudflare Store implementation.
// Options expects CF_API_TOKEN to a cloudflare API token scoped to Workers KV,
// CF_ACCOUNT_ID to contain a string with your cloudflare account ID and
// KV_NAMESPACE_ID to contain the namespace UUID for your KV storage.
func New(opts ...options.Option) (store.Store, error) {
	// Validate Options
	options := options.NewOptions(opts...)
	apiToken, ok := options.Values().Get("CF_API_TOKEN")
	if !ok {
		log.Fatal("Store: No CF_API_TOKEN passed as an option")
	}
	_, ok = apiToken.(string)
	if !ok {
		log.Fatal("Store: Option CF_API_TOKEN contains a non-string")
	}
	accountID, ok := options.Values().Get("CF_ACCOUNT_ID")
	if !ok {
		log.Fatal("Store: No CF_ACCOUNT_ID passed as an option")
	}
	_, ok = accountID.(string)
	if !ok {
		log.Fatal("Store: Option CF_ACCOUNT_ID contains a non-string")
	}
	uuid, ok := options.Values().Get("KV_NAMESPACE_ID")
	if !ok {
		log.Fatal("Store: No KV_NAMESPACE_ID passed as an option")
	}
	_, ok = uuid.(string)
	if !ok {
		log.Fatal("Store: Option KV_NAMESPACE_ID contains a non-string")
	}

	return &workersKV{
		Options:    options,
		httpClient: &http.Client{},
	}, nil
}

// In the cloudflare workers KV implemention, List() doesn't guarantee
// anything as the workers API is eventually consistent.
func (w *workersKV) List() ([]*store.Record, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	accountID, _ := w.Options.Values().Get("CF_ACCOUNT_ID")
	kvID, _ := w.Options.Values().Get("KV_NAMESPACE_ID")

	path := fmt.Sprintf("accounts/%s/storage/kv/namespaces/%s/keys", accountID.(string), kvID.(string))
	response, _, _, err := w.request(ctx, http.MethodGet, path, nil, make(http.Header))
	if err != nil {
		return nil, err
	}
	a := &apiResponse{}
	if err := json.Unmarshal(response, a); err != nil {
		return nil, err
	}
	if !a.Success {
		messages := ""
		for _, m := range a.Errors {
			messages += strconv.Itoa(m.Code) + " " + m.Message + "\n"
		}
		return nil, errors.New(messages)
	}

	var keys []string
	for _, r := range a.Result {
		keys = append(keys, r.Name)
	}
	return w.Read(keys...)
}

func (w *workersKV) Read(keys ...string) ([]*store.Record, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	accountID, _ := w.Options.Values().Get("CF_ACCOUNT_ID")
	kvID, _ := w.Options.Values().Get("KV_NAMESPACE_ID")

	var records []*store.Record
	for _, k := range keys {
		path := fmt.Sprintf("accounts/%s/storage/kv/namespaces/%s/values/%s", accountID.(string), kvID.(string), url.PathEscape(k))
		response, headers, status, err := w.request(ctx, http.MethodGet, path, nil, make(http.Header))
		if err != nil {
			return records, err
		}
		if status < 200 || status >= 300 {
			return records, errors.New("Received unexpected Status " + strconv.Itoa(status) + string(response))
		}
		record := &store.Record{
			Key:   k,
			Value: response,
		}
		if expiry := headers.Get("Expiration"); len(expiry) != 0 {
			expiryUnix, err := strconv.ParseInt(expiry, 10, 64)
			if err != nil {
				return records, err
			}
			record.Expiry = time.Until(time.Unix(expiryUnix, 0))
		}
		records = append(records, record)
	}
	return records, nil
}

func (w *workersKV) Write(records ...*store.Record) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	accountID, _ := w.Options.Values().Get("CF_ACCOUNT_ID")
	kvID, _ := w.Options.Values().Get("KV_NAMESPACE_ID")

	for _, r := range records {
		path := fmt.Sprintf("accounts/%s/storage/kv/namespaces/%s/values/%s", accountID.(string), kvID.(string), url.PathEscape(r.Key))
		if r.Expiry != 0 {
			// Minimum cloudflare TTL is 60 Seconds
			exp := int(math.Max(60, math.Round(r.Expiry.Seconds())))
			path = path + "?expiration_ttl=" + strconv.Itoa(exp)
		}
		headers := make(http.Header)
		resp, _, _, err := w.request(ctx, http.MethodPut, path, r.Value, headers)
		if err != nil {
			return err
		}
		a := &apiResponse{}
		if err := json.Unmarshal(resp, a); err != nil {
			return err
		}
		if !a.Success {
			messages := ""
			for _, m := range a.Errors {
				messages += strconv.Itoa(m.Code) + " " + m.Message + "\n"
			}
			return errors.New(messages)
		}
	}
	return nil
}

func (w *workersKV) Delete(keys ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	accountID, _ := w.Options.Values().Get("CF_ACCOUNT_ID")
	kvID, _ := w.Options.Values().Get("KV_NAMESPACE_ID")

	for _, k := range keys {
		path := fmt.Sprintf("accounts/%s/storage/kv/namespaces/%s/values/%s", accountID.(string), kvID.(string), url.PathEscape(k))
		resp, _, _, err := w.request(ctx, http.MethodDelete, path, nil, make(http.Header))
		if err != nil {
			return err
		}

		a := &apiResponse{}
		if err := json.Unmarshal(resp, a); err != nil {
			return err
		}
		if !a.Success {
			messages := ""
			for _, m := range a.Errors {
				messages += strconv.Itoa(m.Code) + " " + m.Message + "\n"
			}
			return errors.New(messages)
		}
	}
	return nil
}

func (w *workersKV) request(ctx context.Context, method, path string, body interface{}, headers http.Header) ([]byte, http.Header, int, error) {
	var jsonBody []byte
	var err error

	if body != nil {
		if paramBytes, ok := body.([]byte); ok {
			jsonBody = paramBytes
		} else {
			jsonBody, err = json.Marshal(body)
			if err != nil {
				return nil, nil, 0, errors.Wrap(err, "error marshalling params to JSON")
			}
		}
	} else {
		jsonBody = nil
	}
	var reqBody io.Reader
	if jsonBody != nil {
		reqBody = bytes.NewReader(jsonBody)
	}
	req, err := http.NewRequestWithContext(ctx, method, apiBaseURL+path, reqBody)
	for key, value := range headers {
		req.Header[key] = value
	}
	if token, found := w.Options.Values().Get("CF_API_TOKEN"); found {
		req.Header.Set("Authorization", "Bearer "+token.(string))
	}
	req.Header.Set("User-Agent", "micro/1.0 (https://micro.mu)")

	// Official cloudflare client does exponential backoff here
	resp, err := w.httpClient.Do(req)
	if err != nil {
		return nil, nil, 0, err
	}
	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return respBody, resp.Header, resp.StatusCode, err
	}
	return respBody, resp.Header, resp.StatusCode, nil
}

// apiResponse is a cloudflare v4 api response
type apiResponse struct {
	Result []struct {
		ID         string    `json:"id"`
		Type       string    `json:"type"`
		Name       string    `json:"name"`
		Expiration string    `json:"expiration"`
		Content    string    `json:"content"`
		Proxiable  bool      `json:"proxiable"`
		Proxied    bool      `json:"proxied"`
		TTL        int       `json:"ttl"`
		Priority   int       `json:"priority"`
		Locked     bool      `json:"locked"`
		ZoneID     string    `json:"zone_id"`
		ZoneName   string    `json:"zone_name"`
		ModifiedOn time.Time `json:"modified_on"`
		CreatedOn  time.Time `json:"created_on"`
	} `json:"result"`
	Success bool         `json:"success"`
	Errors  []apiMessage `json:"errors"`
	// not sure Messages is ever populated?
	Messages   []apiMessage `json:"messages"`
	ResultInfo struct {
		Page       int `json:"page"`
		PerPage    int `json:"per_page"`
		Count      int `json:"count"`
		TotalCount int `json:"total_count"`
	} `json:"result_info"`
}

// apiMessage is a Cloudflare v4 API Error
type apiMessage struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
