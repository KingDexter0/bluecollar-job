package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func AdminTokenAuth(adminToken string) gin.HandlerFunc {
	return func(c *gin.Context) {
		expected := strings.TrimSpace(adminToken)
		if expected == "" {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, ErrorResponse{
				Error: ErrorBody{
					Code:    "admin_auth_not_configured",
					Message: "admin authentication is not configured",
				},
			})
			return
		}

		token := strings.TrimSpace(c.GetHeader("X-Admin-Token"))
		if token == "" {
			authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
			if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
				token = strings.TrimSpace(authHeader[7:])
			}
		}
		if token != expected {
			c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{
				Error: ErrorBody{
					Code:    "unauthorized",
					Message: "valid admin token is required",
				},
			})
			return
		}

		c.Next()
	}
}
