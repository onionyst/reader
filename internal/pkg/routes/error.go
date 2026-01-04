package routes

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ErrorCode string

const (
	CodeInvalidCredentials ErrorCode = "InvalidCredentials"
	CodeInvalidParameter   ErrorCode = "InvalidParameter"
	CodeInternalServer     ErrorCode = "InternalServer"
	CodeNotFound           ErrorCode = "NotFound"
)

// Error error
type Error struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
}

type ErrorResponse struct {
	Error Error `json:"error"`
}

type AppError struct {
	Status int
	Error
}

func (e *AppError) response() (int, ErrorResponse) {
	return e.Status, ErrorResponse{e.Error}
}

func (e *AppError) errorLog(err error) error {
	if err == nil {
		return fmt.Errorf("%s: %s", e.Code, e.Message)
	}
	return fmt.Errorf("%s: %s (%w)", e.Code, e.Message, err)
}

func AbortWithError(c *gin.Context, appError *AppError, err error) {
	_ = c.Error(appError.errorLog(err))
	c.AbortWithStatusJSON(appError.response())
}

func ErrInvalidCredentials() *AppError {
	return &AppError{
		Status: http.StatusUnauthorized,
		Error: Error{
			Code:    CodeInvalidCredentials,
			Message: "Failed to authorize user credentials.",
		},
	}
}

func ErrInvalidParameter(target string) *AppError {
	return &AppError{
		Status: http.StatusBadRequest,
		Error: Error{
			Code:    CodeInvalidParameter,
			Message: "Failed to retrieve parameter: " + target + ".",
		},
	}
}

func ErrInternalServer() *AppError {
	return &AppError{
		Status: http.StatusInternalServerError,
		Error: Error{
			Code:    CodeInternalServer,
			Message: "Server failed.",
		},
	}
}

func ErrNotFound(target string) *AppError {
	return &AppError{
		Status: http.StatusNotFound,
		Error: Error{
			Code:    CodeNotFound,
			Message: "Failed to find resources: " + target + ".",
		},
	}
}

// InvalidCredentialsError generates invalid credentials error
func InvalidCredentialsError(hint string) (int, map[string]any) {
	if hint != "" {
		hint = fmt.Sprintf(": %s", hint)
	}

	return http.StatusUnauthorized, gin.H{
		"error": Error{
			Code:    "InvalidCredentials",
			Message: fmt.Sprintf("Failed to authorize user credentials%s.", hint),
		},
	}
}

// InvalidParameterError generates an invalid parameter error
func InvalidParameterError(target string) (int, map[string]any) {
	return http.StatusBadRequest, gin.H{
		"error": Error{
			Code:    "InvalidParameter",
			Message: fmt.Sprintf("Failed to retrieve parameter: %s", target),
		},
	}
}

// InternalServerError generates an internal server error
func InternalServerError() (int, map[string]any) {
	return http.StatusInternalServerError, gin.H{
		"error": Error{
			Code:    "InternalServer",
			Message: "Server failed.",
		},
	}
}

// NotFoundError generates a not found error
func NotFoundError(target string) (int, map[string]any) {
	return http.StatusNotFound, gin.H{
		"error": Error{
			Code:    "NotFound",
			Message: fmt.Sprintf("Failed to find resource: %s.", target),
		},
	}
}
