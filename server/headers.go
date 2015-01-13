package server

type Headers interface {
	Add(string, string)
	Del(string)
	Get(string) string
	Set(string, string)
}
