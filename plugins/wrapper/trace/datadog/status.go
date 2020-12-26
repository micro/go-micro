package datadog

import (
	"google.golang.org/grpc/codes"
)

var microCodeToStatusCode = map[int32]codes.Code{
	400: codes.InvalidArgument,
	401: codes.Unauthenticated,
	403: codes.PermissionDenied,
	404: codes.NotFound,
	409: codes.Aborted,
	500: codes.Internal,
}
