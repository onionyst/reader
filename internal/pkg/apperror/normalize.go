package apperror

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-playground/validator/v10"
)

func Normalize(err error) *Error {
	if err == nil {
		return nil
	}

	if ae, ok := As(err); ok {
		ise := InternalServerError(nil)

		if ae.Status == 0 {
			ae.Status = ise.Status
		}
		if ae.Detail.Code == "" {
			ae.Detail.Code = ise.Detail.Code
		}
		if ae.Detail.Message == "" {
			ae.Detail.Message = ise.Detail.Message
		}
		return ae
	}

	// Validation errors from go-playground/validator
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		ae := BadRequest("validation failed").WithInnerCode("ValidationError")
		for _, fe := range ve {
			ae.AddDetailf("ValidationFailed", fmt.Sprintf("failed on '%s'", fe.Tag()), fe.Field())
		}
		return ae
	}

	// JSON binding errors
	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) {
		return BadRequest("malformed JSON").WithCause(err).WithInnerCode("SyntaxError")
	}

	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) {
		return BadRequest("invalid field type").WithCause(err).WithTarget(typeErr.Field).WithInnerCode("TypeError")
	}

	// context errors
	if errors.Is(err, context.DeadlineExceeded) {
		return New(http.StatusGatewayTimeout, "Timeout", "request timed out").WithInnerCode("DeadlineExceeded")
	}

	if errors.Is(err, context.Canceled) {
		return New(http.StatusBadRequest, "Canceled", "request was canceled").WithInnerCode("ContextCanceled")
	}

	return InternalServerError(err).WithInnerCode("UnhandledError")
}
