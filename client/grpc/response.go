package grpc

import (
	"strings"

	"github.com/micro/go-micro/codec"
	"google.golang.org/grpc"
)

type response struct {
	conn *grpc.ClientConn
	stream grpc.ClientStream
	codec grpc.Codec
}

// Read the response
func (r *response) Codec() codec.Reader {
	return nil
}

// read the header
func (r *response) Header() map[string]string {
	md, err := r.stream.Header()
	if err != nil {
		return map[string]string{}
	}
	hdr := make(map[string]string)
	for k, v := range md {
		hdr[k] = strings.Join(v, ",")
	}
	return hdr
}

// Read the undecoded response
func (r *response) Read() ([]byte, error) {
	return nil, nil
}
