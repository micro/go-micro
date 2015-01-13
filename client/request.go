package client

type Request interface {
	Service() string
	Method() string
	ContentType() string
	Request() interface{}
	Headers() Headers
}
