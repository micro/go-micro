package grpc

import (
	"github.com/micro/go-micro/v2/errors"
	"google.golang.org/grpc/status"
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
	if s, ok := status.FromError(err); ok {
		details := s.Details()
		if len(details) == 0 {
			if e := errors.Parse(s.Message()); e.Code > 0 {
				return e // actually a micro error
			}
			return errors.InternalServerError("go.micro.client", s.Message())
		}
		// return first error from details
		return details[0].(error)
	}

	// do nothing
	return err
}
