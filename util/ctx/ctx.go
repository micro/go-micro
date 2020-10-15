package ctx

import (
	"context"
	"net/http"
	"net/textproto"
	"strings"

	"github.com/micro/go-micro/v3/metadata"
)

func FromRequest(r *http.Request) context.Context {
	ctx := r.Context()
	md, ok := metadata.FromContext(ctx)
	if !ok {
		md = make(metadata.Metadata)
	}
	for k, v := range r.Header {
		md[textproto.CanonicalMIMEHeaderKey(k)] = strings.Join(v, ",")
	}
	// pass http host
	md["Host"] = r.Host
	// pass http method
	md["Method"] = r.Method
	return metadata.NewContext(ctx, md)
}
