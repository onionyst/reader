package routes

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Error error
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// InvalidCredentialsError generates invalid credentials error
func InvalidCredentialsError(hint string) (int, map[string]interface{}) {
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
func InvalidParameterError(target string) (int, map[string]interface{}) {
	return http.StatusBadRequest, gin.H{
		"error": Error{
			Code:    "InvalidParameter",
			Message: fmt.Sprintf("Failed to retrieve parameter: %s", target),
		},
	}
}

// InternalServerError generates an internal server error
func InternalServerError() (int, map[string]interface{}) {
	return http.StatusInternalServerError, gin.H{
		"error": Error{
			Code:    "InternalServer",
			Message: "Server failed.",
		},
	}
}

// NotFoundError generates a not found error
func NotFoundError(target string) (int, map[string]interface{}) {
	return http.StatusNotFound, gin.H{
		"error": Error{
			Code:    "NotFound",
			Message: fmt.Sprintf("Failed to find resource: %s.", target),
		},
	}
}
