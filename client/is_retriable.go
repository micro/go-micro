package client

import (
	"github.com/micro/go-micro/errors"
)

type IsRetriableFunc func(err error) bool

// always retry on error
func AlwaysRetry(err error) bool {
	return true
}

func Only500Errors(err error) bool {
	errorData := errors.Parse(err.Error())

	if(errorData.Code >= 500) {
		return true
	}

	return false
}