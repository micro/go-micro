package server

type Request interface {
	Headers() Headers
	Session(string) string
}
