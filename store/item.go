package store

type Item interface {
	Key() string
	Value() []byte
}
