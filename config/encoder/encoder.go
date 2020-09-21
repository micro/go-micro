// Package encoder handles source encoding formats
package encoder

type Encoder interface {
	Encode(interface{}) ([]byte, error)
	Decode([]byte, interface{}) error
	String() string
}
