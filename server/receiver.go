package server

type Receiver interface {
	Name() string
	Handler() interface{}
}
