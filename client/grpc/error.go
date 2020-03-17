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
	return errors.InternalServerError("go.micro.client", s.Message())
}
