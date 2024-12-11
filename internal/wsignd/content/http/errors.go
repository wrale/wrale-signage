package http

import "net/http"

type HTTPError interface {
	error
	StatusCode() int
}

type httpError struct {
	msg  string
	code int
}

func (e *httpError) Error() string {
	return e.msg
}

func (e *httpError) StatusCode() int {
	return e.code
}

func ErrInvalidRequest(msg string) error {
	return &httpError{msg: msg, code: http.StatusBadRequest}
}

func ErrNotFound(msg string) error {
	return &httpError{msg: msg, code: http.StatusNotFound}
}

func ErrConflict(msg string) error {
	return &httpError{msg: msg, code: http.StatusConflict}
}
