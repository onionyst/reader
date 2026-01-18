package middleware

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"reader/internal/pkg/apperror"
)

const (
	loggerKey = "logger_entry"
)

func RequestLogger(log *logrus.Logger, skipPaths ...string) gin.HandlerFunc {
	skip := make(map[string]struct{}, len(skipPaths))
	for _, p := range skipPaths {
		skip[p] = struct{}{}
	}

	return func(c *gin.Context) {
		if _, ok := skip[c.Request.URL.Path]; ok {
			c.Next()
			return
		}

		start := time.Now()
		rid := GetRequestID(c)

		entry := logrus.NewEntry(log).WithFields(logrus.Fields{
			"request_id": rid,
			"method":     c.Request.Method,
			"path":       c.Request.URL.Path,
			"full_path":  c.FullPath(),
			"client_ip":  c.ClientIP(),
		})

		c.Set(loggerKey, entry)

		ctx := context.WithValue(c.Request.Context(), loggerKey, entry)
		c.Request = c.Request.WithContext(ctx)

		c.Next()

		status := c.Writer.Status()
		lat := time.Since(start)

		entry = entry.WithFields(logrus.Fields{
			"status":     status,
			"latency_ms": lat.Milliseconds(),
		})

		if v, ok := c.Get(AppErrorKey); ok {
			if ae, ok := v.(*apperror.Error); ok && ae != nil {
				entry = entry.WithFields(logrus.Fields{
					"code":   ae.Detail.Code,
					"target": ae.Detail.Target,
				})
				if ae.Cause != nil {
					entry = entry.WithError(ae.Cause)
				} else {
					entry = entry.WithError(ae)
				}
			}
		} else if len(c.Errors) > 0 {
			entry = entry.WithError(c.Errors.Last().Err)
		}

		if status >= 400 {
			query := c.Request.URL.Query()
			if len(query) > 0 {
				entry = entry.WithField("query", query.Encode())
			}
		}

		switch {
		case status >= 500:
			entry.Error("request failed")
		case status >= 400:
			entry.Warn("request rejected")
		default:
			entry.Info("request completed")
		}
	}
}

func GetLogger(c *gin.Context, fallback *logrus.Logger) *logrus.Entry {
	if v, ok := c.Get(loggerKey); ok {
		if e, ok := v.(*logrus.Entry); ok && e != nil {
			return e
		}
	}
	return logrus.NewEntry(fallback)
}
