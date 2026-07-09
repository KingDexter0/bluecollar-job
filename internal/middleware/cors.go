package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func CORS(allowedOrigins []string, allowLocalhost bool) gin.HandlerFunc {
	allowed := map[string]bool{}
	for _, origin := range allowedOrigins {
		allowed[origin] = true
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" && (allowed[origin] || (allowLocalhost && isLocalhostOrigin(origin))) {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID, X-Admin-Token")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func isLocalhostOrigin(origin string) bool {
	return len(origin) >= len("http://localhost:") && origin[:len("http://localhost:")] == "http://localhost:" ||
		len(origin) >= len("http://127.0.0.1:") && origin[:len("http://127.0.0.1:")] == "http://127.0.0.1:"
}
