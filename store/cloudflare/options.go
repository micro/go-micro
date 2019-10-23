package cloudflare

import (
	"github.com/micro/go-micro/config/options"
)

// Token sets the cloudflare api token
func ApiToken(t string) options.Option {
	// TODO: change to store.cf.api_token
	return options.WithValue("CF_API_TOKEN", t)
}

// AccountID sets the cloudflare account id
func AccountID(id string) options.Option {
	// TODO: change to store.cf.account_id
	return options.WithValue("CF_ACCOUNT_ID", id)
}

// Namespace sets the KV namespace
func Namespace(ns string) options.Option {
	// TODO: change to store.cf.namespace
	return options.WithValue("KV_NAMESPACE_ID", ns)
}
