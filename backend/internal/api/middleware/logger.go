// Package middleware provides Gin middleware for the PUSINGBERAT API server.
package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// Logger returns a Gin middleware that logs every request using Go's
// structured slog package. It includes the X-Request-ID if the RequestID
// middleware has been applied earlier in the chain.
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// Process request.
		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		attrs := []any{
			"status", status,
			"method", c.Request.Method,
			"path", path,
			"latency_ms", latency.Milliseconds(),
			"ip", c.ClientIP(),
		}
		if rid, exists := c.Get("request_id"); exists {
			attrs = append(attrs, "request_id", rid)
		}
		if query != "" {
			attrs = append(attrs, "query", query)
		}
		if len(c.Errors) > 0 {
			attrs = append(attrs, "errors", c.Errors.String())
		}

		switch {
		case status >= 500:
			slog.Error("request completed", attrs...)
		case status >= 400:
			slog.Warn("request completed", attrs...)
		default:
			slog.Info("request completed", attrs...)
		}
	}
}
