package apperror

import (
	"errors"
	"net/http"
)

type ErrorResponse struct {
	Error *ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    string        `json:"code"`
	Message string        `json:"message"`
	Target  string        `json:"target,omitempty"`
	Details []ErrorDetail `json:"details,omitempty"`
	Inner   *InnerError   `json:"innererror,omitempty"`
}

type InnerError struct {
	Code  string      `json:"code,omitempty"`
	Inner *InnerError `json:"innererror,omitempty"`
}

type Error struct {
	Status int
	Detail ErrorDetail
	Cause  error
}

func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Cause != nil {
		return e.Detail.Code + ": " + e.Cause.Error()
	}
	return e.Detail.Code + ": " + e.Detail.Message
}

func (e *Error) Unwrap() error {
	return e.Cause
}

func New(stats int, code, message string) *Error {
	return &Error{
		Status: stats,
		Detail: ErrorDetail{
			Code:    code,
			Message: message,
		},
	}
}

func (e *Error) WithCause(cause error) *Error {
	e.Cause = cause
	return e
}

func (e *Error) WithTarget(target string) *Error {
	e.Detail.Target = target
	return e
}

func (e *Error) AddDetail(d ErrorDetail) *Error {
	e.Detail.Details = append(e.Detail.Details, d)
	return e
}

func (e *Error) AddDetailf(code, message, target string) *Error {
	return e.AddDetail(ErrorDetail{
		Code:    code,
		Message: message,
		Target:  target,
	})
}

func (e *Error) WithInnerCode(code string) *Error {
	if e.Detail.Inner == nil {
		e.Detail.Inner = &InnerError{}
	}
	e.Detail.Inner.Code = code
	return e
}

func (e *Error) Response() ErrorResponse {
	if e == nil {
		return InternalServerError(nil).Response()
	}
	return ErrorResponse{
		Error: &e.Detail,
	}
}

func As(err error) (*Error, bool) {
	var e *Error
	if errors.As(err, &e) {
		return e, true
	}
	return nil, false
}

func BadRequest(message string) *Error {
	return New(http.StatusBadRequest, "BadRequest", message)
}

func Unauthorized(message string) *Error {
	return New(http.StatusUnauthorized, "Unauthorized", message)
}

func NotFound(message string) *Error {
	return New(http.StatusNotFound, "NotFound", message)
}

func NotFoundTarget(target string) *Error {
	return NotFound("resource not found").WithTarget(target)
}

func Conflict(message string) *Error {
	return New(http.StatusConflict, "Conflict", message)
}

func InternalServerError(cause error) *Error {
	return New(http.StatusInternalServerError, "InternalServerError", "internal server error").WithCause(cause)
}
