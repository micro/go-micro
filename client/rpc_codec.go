package client

import (
	errs "errors"

	"go-micro.dev/v6/internal/mucp"
)

const (
	// lastStreamResponseError is sent by the server to signal the end of a
	// streaming response.
	lastStreamResponseError = "EOS"
)

// serverError represents an error that has been returned from
// the remote side of the RPC connection.
type serverError string

func (e serverError) Error() string { return string(e) }

// errShutdown holds the specific error for closing/closed connections.
var (
	errShutdown = errs.New("connection is shut down")
)

// DefaultContentType is the default content type for outbound requests.
const DefaultContentType = "application/json"

// DefaultCodecs is the default codec map. The mucp wire framing lives in
// go-micro.dev/v6/internal/mucp.
var DefaultCodecs = mucp.DefaultCodecs
