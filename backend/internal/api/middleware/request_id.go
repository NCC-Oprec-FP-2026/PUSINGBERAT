package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const requestIDHeader = "X-Request-ID"

// RequestID returns a Gin middleware that ensures every request has a unique
// X-Request-ID header. If the client sends one it is reused; otherwise a new
// UUID v4 is generated. The ID is also set on the response headers so that
// clients and log aggregators can correlate requests.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.GetHeader(requestIDHeader)
		if rid == "" {
			rid = uuid.New().String()
		}

		// Store in context so slog / handlers can read it.
		c.Set("request_id", rid)

		// Echo back on the response.
		c.Header(requestIDHeader, rid)

		c.Next()
	}
}
