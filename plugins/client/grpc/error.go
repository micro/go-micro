package grpc

import (
	"github.com/asim/go-micro/v3/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net/http"
)

func microError(err error) error {
	// no error
	switch err {
	case nil:
		return nil
	}

	if verr, ok := err.(*errors.Error); ok {
		return verr
	}

	// grpc error
	s, ok := status.FromError(err)
	if !ok {
		return err
	}

	// return first error from details
	if details := s.Details(); len(details) > 0 {
		return microError(details[0].(error))
	}

	// try to decode micro *errors.Error
	if e := errors.Parse(s.Message()); e.Code > 0 {
		return e // actually a micro error
	}

	// fallback
	return errors.New("go.micro.client", s.Message(), microStatusFromGrpcCode(s.Code()))
}

func microStatusFromGrpcCode(code codes.Code) int32 {
	switch code {
	case codes.OK:
		return http.StatusOK
	case codes.InvalidArgument:
		return http.StatusBadRequest
	case codes.DeadlineExceeded:
		return http.StatusRequestTimeout
	case codes.NotFound:
		return http.StatusNotFound
	case codes.AlreadyExists:
		return http.StatusConflict
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	case codes.FailedPrecondition:
		return http.StatusPreconditionFailed
	case codes.Unimplemented:
		return http.StatusNotImplemented
	case codes.Internal:
		return http.StatusInternalServerError
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	}

	return http.StatusInternalServerError
}