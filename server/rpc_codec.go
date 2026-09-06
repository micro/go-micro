package server

import "go-micro.dev/v6/internal/mucp"

// DefaultContentType is the default codec content type.
const DefaultContentType = "application/protobuf"

// DefaultCodecs is the default codec map. The mucp wire framing lives in
// go-micro.dev/v6/internal/mucp.
var DefaultCodecs = mucp.DefaultCodecs

// getHeader reads a header with the canonical X- prefix fallback.
func getHeader(hdr string, md map[string]string) string {
	return mucp.GetHeader(hdr, md)
}
