package codec

// Codec is used for encoding where the transport doesn't natively support
// headers in the message type. In this case the entire message is
// encoded as the payload
type Codec interface {
	Marshal(interface{}) ([]byte, error)
	Unmarshal([]byte, interface{}) error
	String() string
}
