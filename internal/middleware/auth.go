package middleware

import (
	"net/http"
	"strings"

	"bluecollarjob/internal/service"

	"github.com/gin-gonic/gin"
)

const EmployerIDKey = "employer_id"

func EmployerAuth(auth service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{Error: ErrorBody{
				Code:    "missing_token",
				Message: "Authorization bearer token is required",
			}})
			return
		}

		token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
		if token == "" || token == header {
			c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{Error: ErrorBody{
				Code:    "invalid_token",
				Message: "Authorization header must use Bearer token",
			}})
			return
		}

		employerID, err := auth.ParseEmployerToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{Error: ErrorBody{
				Code:    "invalid_token",
				Message: "invalid or expired token",
			}})
			return
		}

		c.Set(EmployerIDKey, employerID)
		c.Next()
	}
}

func GetEmployerID(c *gin.Context) string {
	value, _ := c.Get(EmployerIDKey)
	employerID, _ := value.(string)
	return employerID
}
