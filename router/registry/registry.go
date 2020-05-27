package registry

import "github.com/micro/go-micro/v2/router"

// NewRouter returns an initialised registry router
func NewRouter(opts ...router.Option) router.Router {
	return router.NewRouter(opts...)
}
