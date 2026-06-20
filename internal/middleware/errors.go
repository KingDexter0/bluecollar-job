package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func JSONErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) == 0 || c.Writer.Written() {
			return
		}

		c.JSON(c.Writer.Status(), ErrorResponse{
			Error: ErrorBody{
				Code:    "request_error",
				Message: c.Errors.Last().Error(),
			},
		})
	}
}

func JSONRecovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{
			Error: ErrorBody{
				Code:    "internal_server_error",
				Message: "internal server error",
			},
		})
	})
}
