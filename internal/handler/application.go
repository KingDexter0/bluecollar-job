package handler

import (
	"net/http"

	"bluecollarjob/internal/service"

	"github.com/gin-gonic/gin"
)

type ApplicationHandler struct {
	applications service.ApplicationService
}

func NewApplicationHandler(applications service.ApplicationService) *ApplicationHandler {
	return &ApplicationHandler{applications: applications}
}

type createApplicationRequest struct {
	UserID string `json:"user_id"`
	JobID  string `json:"job_id"`
	Source string `json:"source"`
}

func (h *ApplicationHandler) Create(c *gin.Context) {
	var request createApplicationRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_json", "invalid JSON body")
		return
	}

	request.UserID = requiredString(request.UserID)
	request.JobID = requiredString(request.JobID)
	request.Source = requiredString(request.Source)

	if request.UserID == "" {
		writeError(c, http.StatusBadRequest, "missing_user_id", "user_id is required")
		return
	}
	if request.JobID == "" {
		writeError(c, http.StatusBadRequest, "missing_job_id", "job_id is required")
		return
	}

	application, err := h.applications.CreateApplication(c.Request.Context(), service.CreateApplicationInput{
		UserID: request.UserID,
		JobID:  request.JobID,
		Source: request.Source,
	})
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"application": newApplicationResponse(*application)})
}

func (h *ApplicationHandler) GetByID(c *gin.Context) {
	application, err := h.applications.GetApplicationByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"application": newApplicationResponse(*application)})
}

func (h *ApplicationHandler) ListByWorker(c *gin.Context) {
	limit, offset := parsePagination(c)
	applications, err := h.applications.ListApplicationsByUser(c.Request.Context(), c.Param("id"), limit, offset)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	response := make([]applicationResponse, 0, len(applications))
	for _, application := range applications {
		response = append(response, newApplicationResponse(application))
	}

	c.JSON(http.StatusOK, gin.H{"applications": response})
}
