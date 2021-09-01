package ctx

import (
	"context"
	"net/http"
	"strings"

	"github.com/chinahtl/go-micro/v3/metadata"
)

func FromRequest(r *http.Request) context.Context {
	ctx := context.Background()
	md := make(metadata.Metadata)
	for k, v := range r.Header {
		md[k] = strings.Join(v, ",")
	}
	return metadata.NewContext(ctx, md)
}
