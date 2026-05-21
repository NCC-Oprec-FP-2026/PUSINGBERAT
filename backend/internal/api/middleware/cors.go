package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORSConfig holds the allowed origins for CORS.
type CORSConfig struct {
	AllowedOrigins []string
}

// CORS returns a Gin middleware that sets Cross-Origin Resource Sharing
// headers. It handles preflight OPTIONS requests automatically.
func CORS(cfg CORSConfig) gin.HandlerFunc {
	// Build the origins lookup for O(1) checking.
	originSet := make(map[string]bool, len(cfg.AllowedOrigins))
	for _, o := range cfg.AllowedOrigins {
		originSet[strings.TrimSpace(o)] = true
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// If the request origin is in the allow-list, reflect it back.
		if originSet[origin] {
			c.Header("Access-Control-Allow-Origin", origin)
		} else if originSet["*"] {
			c.Header("Access-Control-Allow-Origin", "*")
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Request-ID")
		c.Header("Access-Control-Expose-Headers", "X-Request-ID")
		c.Header("Access-Control-Max-Age", "86400")

		// Handle preflight.
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
