package errorsx

import (
	"errors"
	"fmt"
	"net/http"
)

var (
	ErrNotFound     = errors.New("resource not found")
	ErrConflict     = errors.New("resource conflict")
	ErrInvalid      = errors.New("invalid input")
	ErrUnavailable  = errors.New("upstream unavailable")
	ErrCircuitOpen  = errors.New("circuit breaker open")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
	ErrRateLimited  = errors.New("rate limited")
)

type Coded struct {
	Code    int
	HTTP    int
	Message string
	Cause   error
}

func (e *Coded) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *Coded) Unwrap() error { return e.Cause }

func New(httpStatus, code int, msg string) *Coded {
	return &Coded{HTTP: httpStatus, Code: code, Message: msg}
}

func Wrap(httpStatus, code int, msg string, err error) *Coded {
	return &Coded{HTTP: httpStatus, Code: code, Message: msg, Cause: err}
}

func HTTPStatus(err error) int {
	var c *Coded
	if errors.As(err, &c) {
		return c.HTTP
	}
	switch {
	case errors.Is(err, ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrConflict):
		return http.StatusConflict
	case errors.Is(err, ErrInvalid):
		return http.StatusBadRequest
	case errors.Is(err, ErrUnauthorized):
		return http.StatusUnauthorized
	case errors.Is(err, ErrForbidden):
		return http.StatusForbidden
	case errors.Is(err, ErrRateLimited):
		return http.StatusTooManyRequests
	case errors.Is(err, ErrCircuitOpen), errors.Is(err, ErrUnavailable):
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

func BizCode(err error) int {
	var c *Coded
	if errors.As(err, &c) {
		return c.Code
	}
	switch {
	case errors.Is(err, ErrNotFound):
		return 40401
	case errors.Is(err, ErrConflict):
		return 40901
	case errors.Is(err, ErrInvalid):
		return 40001
	case errors.Is(err, ErrUnauthorized):
		return 40101
	case errors.Is(err, ErrForbidden):
		return 40301
	case errors.Is(err, ErrRateLimited):
		return 42901
	case errors.Is(err, ErrCircuitOpen):
		return 50301
	case errors.Is(err, ErrUnavailable):
		return 50202
	default:
		return 50001
	}
}
