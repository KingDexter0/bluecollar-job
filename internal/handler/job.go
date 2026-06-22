package handler

import (
	"net/http"

	"bluecollarjob/internal/service"

	"github.com/gin-gonic/gin"
)

type JobHandler struct {
	jobs service.JobService
}

func NewJobHandler(jobs service.JobService) *JobHandler {
	return &JobHandler{jobs: jobs}
}

func (h *JobHandler) ListActive(c *gin.Context) {
	limit, offset := parsePagination(c)
	jobs, err := h.jobs.ListActiveJobs(c.Request.Context(), limit, offset)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	response := make([]jobResponse, 0, len(jobs))
	for _, job := range jobs {
		response = append(response, newJobResponse(job))
	}

	c.JSON(http.StatusOK, gin.H{"jobs": response})
}

func (h *JobHandler) GetByID(c *gin.Context) {
	job, err := h.jobs.GetJobByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"job": newJobResponse(*job)})
}
