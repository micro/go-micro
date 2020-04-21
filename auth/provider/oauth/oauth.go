package oauth

import (
	"fmt"
	"net/url"

	"github.com/micro/go-micro/v2/auth/provider"
)

// NewProvider returns an initialised oauth provider
func NewProvider(opts ...provider.Option) provider.Provider {
	var options provider.Options
	for _, o := range opts {
		o(&options)
	}
	return &oauth{options}
}

type oauth struct {
	opts provider.Options
}

func (o *oauth) String() string {
	return "oauth"
}

func (o *oauth) Options() provider.Options {
	return o.opts
}

func (o *oauth) Endpoint(opts ...provider.EndpointOption) string {
	var options provider.EndpointOptions
	for _, o := range opts {
		o(&options)
	}

	params := make(url.Values)
	params.Add("response_type", "code")

	if len(options.State) > 0 {
		params.Add("state", options.State)
	}

	if len(options.LoginHint) > 0 {
		params.Add("login_hint", options.LoginHint)
	}

	if clientID := o.opts.ClientID; len(clientID) > 0 {
		params.Add("client_id", clientID)
	}

	if scope := o.opts.Scope; len(scope) > 0 {
		params.Add("scope", scope)
	}

	if redir := o.Redirect(); len(redir) > 0 {
		params.Add("redirect_uri", redir)
	}

	return fmt.Sprintf("%v?%v", o.opts.Endpoint, params.Encode())
}

func (o *oauth) Redirect() string {
	return o.opts.Redirect
}
