package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"reader/internal/pkg/apperror"
)

const (
	AppErrorKey = "app_error"
	RawErrorKey = "raw_error"
)

func ErrorResponder(log *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if c.Writer.Written() {
			return
		}
		if len(c.Errors) == 0 {
			return
		}

		rawErr := c.Errors.Last().Err
		ae := apperror.Normalize(rawErr)

		c.Set(AppErrorKey, ae)
		c.Set(RawErrorKey, rawErr)

		c.AbortWithStatusJSON(ae.Status, ae.Response())
	}
}
