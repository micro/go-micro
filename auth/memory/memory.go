package memory

import "github.com/micro/go-micro/v2/auth"

// NewAuth returns an instance of memory auth
func NewAuth(opts ...auth.Option) auth.Auth {
	return auth.DefaultAuth
}
