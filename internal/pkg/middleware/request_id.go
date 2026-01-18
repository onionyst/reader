package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"

	"github.com/gin-gonic/gin"
)

const (
	requestIDHeader = "X-Request-Id"
	requestIDKey    = "request_id"
)

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.Request.Header.Get(requestIDHeader)
		if rid == "" {
			rid = newRequestID()
		}

		c.Set(requestIDKey, rid)

		ctx := context.WithValue(c.Request.Context(), requestIDKey, rid)
		c.Request = c.Request.WithContext(ctx)

		c.Writer.Header().Set(requestIDHeader, rid)
		c.Next()
	}
}

func GetRequestID(c *gin.Context) string {
	if v, ok := c.Get(requestIDKey); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func newRequestID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
