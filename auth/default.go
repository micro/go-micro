package auth

var (
	DefaultAuth = NewAuth()
)

// NewAuth returns a new default registry which is memory
func NewAuth(opts ...Option) Auth {
	return newRegistry(opts...)
}
