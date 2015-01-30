package errors

import (
	"encoding/json"
	"net/http"
)

type Error struct {
	Id     string `json:"id"`
	Code   int32  `json:"code"`
	Detail string `json:"detail"`
	Status string `json:"status"`
}

func (e *Error) Error() string {
	b, _ := json.Marshal(e)
	return string(b)
}

func New(id, detail string, code int32) error {
	return &Error{
		Id:     id,
		Code:   code,
		Detail: detail,
		Status: http.StatusText(int(code)),
	}
}

func Parse(err string) *Error {
	e := new(Error)
	errr := json.Unmarshal([]byte(err), e)
	if errr != nil {
		e.Detail = err
	}
	return e
}

func BadRequest(id, detail string) error {
	return &Error{
		Id:     id,
		Code:   400,
		Detail: detail,
		Status: http.StatusText(400),
	}
}

func Unauthorized(id, detail string) error {
	return &Error{
		Id:     id,
		Code:   401,
		Detail: detail,
		Status: http.StatusText(401),
	}
}

func Forbidden(id, detail string) error {
	return &Error{
		Id:     id,
		Code:   403,
		Detail: detail,
		Status: http.StatusText(403),
	}
}

func NotFound(id, detail string) error {
	return &Error{
		Id:     id,
		Code:   404,
		Detail: detail,
		Status: http.StatusText(404),
	}
}

func InternalServerError(id, detail string) error {
	return &Error{
		Id:     id,
		Code:   500,
		Detail: detail,
		Status: http.StatusText(500),
	}
}
