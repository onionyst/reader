package middleware

import (
	"fmt"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"reader/internal/pkg/apperror"
)

func Recovery(log *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				entry := GetLogger(c, log)
				entry.WithFields(logrus.Fields{
					"panic": fmt.Sprintf("%v", rec),
					"stack": string(debug.Stack()),
				}).Error("panic recovered")

				_ = c.Error(apperror.InternalServerError(fmt.Errorf("panic: %v", rec)).WithInnerCode("Panic"))
				c.Abort()
			}
		}()
		c.Next()
	}
}
