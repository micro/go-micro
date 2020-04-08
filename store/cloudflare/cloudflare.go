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
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/micro/go-micro/v2/store"
	"github.com/pkg/errors"

	"github.com/patrickmn/go-cache"
)

const (
	apiBaseURL = "https://api.cloudflare.com/client/v4/"
)

type workersKV struct {
	options store.Options
	// cf account id
	account string
	// cf api token
	token string
	// cf kv namespace
	namespace string
	// http client to use
	httpClient *http.Client
	// cache
	cache *cache.Cache
}

// apiResponse is a cloudflare v4 api response
type apiResponse struct {
	Result []struct {
		ID         string    `json:"id"`
		Type       string    `json:"type"`
		Name       string    `json:"name"`
		Expiration int64     `json:"expiration"`
		Content    string    `json:"content"`
		Proxiable  bool      `json:"proxiable"`
		Proxied    bool      `json:"proxied"`
		TTL        int64     `json:"ttl"`
		Priority   int64     `json:"priority"`
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

// getOptions returns account id, token and namespace
func getOptions() (string, string, string) {
	accountID := strings.TrimSpace(os.Getenv("CF_ACCOUNT_ID"))
	apiToken := strings.TrimSpace(os.Getenv("CF_API_TOKEN"))
	namespace := strings.TrimSpace(os.Getenv("KV_NAMESPACE_ID"))

	return accountID, apiToken, namespace
}

func validateOptions(account, token, namespace string) {
	if len(account) == 0 {
		log.Fatal("Store: CF_ACCOUNT_ID is blank")
	}

	if len(token) == 0 {
		log.Fatal("Store: CF_API_TOKEN is blank")
	}

	if len(namespace) == 0 {
		log.Fatal("Store: KV_NAMESPACE_ID is blank")
	}
}

func (w *workersKV) Close() error {
	return nil
}

func (w *workersKV) Init(opts ...store.Option) error {
	for _, o := range opts {
		o(&w.options)
	}
	if len(w.options.Database) > 0 {
		w.namespace = w.options.Database
	}
	if w.options.Context == nil {
		w.options.Context = context.TODO()
	}
	ttl := w.options.Context.Value("STORE_CACHE_TTL")
	if ttl != nil {
		ttlduration, ok := ttl.(time.Duration)
		if !ok {
			log.Fatal("STORE_CACHE_TTL from context must be type int64")
		}
		w.cache = cache.New(ttlduration, 3*ttlduration)
	}
	return nil
}

func (w *workersKV) list(prefix string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	path := fmt.Sprintf("accounts/%s/storage/kv/namespaces/%s/keys", w.account, w.namespace)

	body := make(map[string]string)

	if len(prefix) > 0 {
		body["prefix"] = prefix
	}

	response, _, _, err := w.request(ctx, http.MethodGet, path, body, make(http.Header))
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

	keys := make([]string, 0, len(a.Result))

	for _, r := range a.Result {
		keys = append(keys, r.Name)
	}

	return keys, nil
}

// In the cloudflare workers KV implemention, List() doesn't guarantee
// anything as the workers API is eventually consistent.
func (w *workersKV) List(opts ...store.ListOption) ([]string, error) {
	keys, err := w.list("")
	if err != nil {
		return nil, err
	}

	return keys, nil
}

func (w *workersKV) Read(key string, opts ...store.ReadOption) ([]*store.Record, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var options store.ReadOptions
	for _, o := range opts {
		o(&options)
	}

	keys := []string{key}

	if options.Prefix {
		k, err := w.list(key)
		if err != nil {
			return nil, err
		}
		keys = k
	}

	//nolint:prealloc
	var records []*store.Record

	for _, k := range keys {
		if w.cache != nil {
			if resp, hit := w.cache.Get(k); hit {
				if record, ok := resp.(*store.Record); ok {
					records = append(records, record)
					continue
				}
			}
		}

		path := fmt.Sprintf("accounts/%s/storage/kv/namespaces/%s/values/%s", w.account, w.namespace, url.PathEscape(k))
		response, headers, status, err := w.request(ctx, http.MethodGet, path, nil, make(http.Header))
		if err != nil {
			return records, err
		}
		if status < 200 || status >= 300 {
			if status == 404 {
				return nil, store.ErrNotFound
			}

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
		if w.cache != nil {
			w.cache.Set(record.Key, record, cache.DefaultExpiration)
		}
		records = append(records, record)
	}

	return records, nil
}

func (w *workersKV) Write(r *store.Record, opts ...store.WriteOption) error {
	// Set it in local cache, with the global TTL from options
	if w.cache != nil {
		w.cache.Set(r.Key, r, cache.DefaultExpiration)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	path := fmt.Sprintf("accounts/%s/storage/kv/namespaces/%s/values/%s", w.account, w.namespace, url.PathEscape(r.Key))
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

	return nil
}

func (w *workersKV) Delete(key string, opts ...store.DeleteOption) error {
	if w.cache != nil {
		w.cache.Delete(key)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	path := fmt.Sprintf("accounts/%s/storage/kv/namespaces/%s/values/%s", w.account, w.namespace, url.PathEscape(key))
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
	if err != nil {
		return nil, nil, 0, errors.Wrap(err, "error creating new request")
	}

	for key, value := range headers {
		req.Header[key] = value
	}

	// set token if it exists
	if len(w.token) > 0 {
		req.Header.Set("Authorization", "Bearer "+w.token)
	}

	// set the user agent to micro
	req.Header.Set("User-Agent", "micro/1.0 (https://micro.mu)")

	// Official cloudflare client does exponential backoff here
	// TODO: retry and use util/backoff
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

func (w *workersKV) String() string {
	return "cloudflare"
}

func (w *workersKV) Options() store.Options {
	return w.options
}

// NewStore returns a cloudflare Store implementation.
// Account ID, Token and Namespace must either be passed as options or
// environment variables. If set as env vars we expect the following;
// CF_API_TOKEN to a cloudflare API token scoped to Workers KV.
// CF_ACCOUNT_ID to contain a string with your cloudflare account ID.
// KV_NAMESPACE_ID to contain the namespace UUID for your KV storage.
func NewStore(opts ...store.Option) store.Store {
	var options store.Options
	for _, o := range opts {
		o(&options)
	}

	// get options from environment
	account, token, namespace := getOptions()

	if len(account) == 0 {
		account = getAccount(options.Context)
	}

	if len(token) == 0 {
		token = getToken(options.Context)
	}

	if len(namespace) == 0 {
		namespace = options.Database
	}

	// validate options are not blank or log.Fatal
	validateOptions(account, token, namespace)

	return &workersKV{
		account:    account,
		namespace:  namespace,
		token:      token,
		options:    options,
		httpClient: &http.Client{},
	}
}
