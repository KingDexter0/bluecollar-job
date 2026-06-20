package handler

import (
	"net/http"

	"bluecollarjob/internal/service"

	"github.com/gin-gonic/gin"
)

type HealthHandler struct {
	healthService *service.HealthService
}

func NewHealthHandler(healthService *service.HealthService) *HealthHandler {
	return &HealthHandler{healthService: healthService}
}

func (h *HealthHandler) Check(c *gin.Context) {
	response := h.healthService.Check(c.Request.Context())
	statusCode := http.StatusOK
	if response.Status != "ok" {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, response)
}
