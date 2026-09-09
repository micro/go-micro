// Package errors provides a way to return detailed information
// for an RPC request error. The error is normally JSON encoded.
package errors

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

//go:generate protoc -I. --go_out=paths=source_relative:. errors.proto

func (e *Error) Error() string {
	b, _ := json.Marshal(e)
	return string(b)
}

// New generates a custom error.
func New(id, detail string, code int32) error {
	return &Error{
		Id:     id,
		Code:   code,
		Detail: detail,
		Status: http.StatusText(int(code)),
	}
}

// Parse tries to parse a JSON string into an error. If that
// fails, it will set the given string as the error detail.
func Parse(err string) *Error {
	e := new(Error)
	errr := json.Unmarshal([]byte(err), e)
	if errr != nil {
		e.Detail = err
	}
	return e
}

func newError(id string, code int32, detail string, a ...interface{}) error {
	if len(a) > 0 {
		detail = fmt.Sprintf(detail, a...)
	}
	return &Error{
		Id:     id,
		Code:   code,
		Detail: detail,
		Status: http.StatusText(int(code)),
	}
}

// BadRequest generates a 400 error.
func BadRequest(id, format string, a ...interface{}) error {
	return newError(id, 400, format, a...)
}

// Unauthorized generates a 401 error.
func Unauthorized(id, format string, a ...interface{}) error {
	return newError(id, 401, format, a...)
}

// Forbidden generates a 403 error.
func Forbidden(id, format string, a ...interface{}) error {
	return newError(id, 403, format, a...)
}

// NotFound generates a 404 error.
func NotFound(id, format string, a ...interface{}) error {
	return newError(id, 404, format, a...)
}

// MethodNotAllowed generates a 405 error.
func MethodNotAllowed(id, format string, a ...interface{}) error {
	return newError(id, 405, format, a...)
}

// Timeout generates a 408 error.
func Timeout(id, format string, a ...interface{}) error {
	return newError(id, 408, format, a...)
}

// Conflict generates a 409 error.
func Conflict(id, format string, a ...interface{}) error {
	return newError(id, 409, format, a...)
}

// InternalServerError generates a 500 error.
func InternalServerError(id, format string, a ...interface{}) error {
	return newError(id, 500, format, a...)
}

// Equal tries to compare errors.
func Equal(err1 error, err2 error) bool {
	verr1, ok1 := err1.(*Error)
	verr2, ok2 := err2.(*Error)

	if ok1 != ok2 {
		return false
	}

	if !ok1 {
		return err1 == err2
	}

	if verr1.Code != verr2.Code {
		return false
	}

	return true
}

// FromError try to convert go error to *Error.
func FromError(err error) *Error {
	if err == nil {
		return nil
	}
	if verr, ok := err.(*Error); ok && verr != nil {
		return verr
	}

	return Parse(err.Error())
}

// As finds the first error in err's chain that matches *Error.
func As(err error) (*Error, bool) {
	if err == nil {
		return nil, false
	}
	var merr *Error
	if errors.As(err, &merr) {
		return merr, true
	}
	return nil, false
}

func NewMultiError() *MultiError {
	return &MultiError{
		Errors: make([]*Error, 0),
	}
}

func (e *MultiError) Append(err ...*Error) {
	e.Errors = append(e.Errors, err...)
}

func (e *MultiError) HasErrors() bool {
	return len(e.Errors) > 0
}

func (e *MultiError) Error() string {
	b, _ := json.Marshal(e)
	return string(b)
}
