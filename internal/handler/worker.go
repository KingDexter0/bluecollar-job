package handler

import (
	"net/http"

	"bluecollarjob/internal/service"

	"github.com/gin-gonic/gin"
)

type WorkerHandler struct {
	workers service.WorkerService
}

func NewWorkerHandler(workers service.WorkerService) *WorkerHandler {
	return &WorkerHandler{workers: workers}
}

type createWorkerRequest struct {
	PhoneNumber        string  `json:"phone_number"`
	FullName           string  `json:"full_name"`
	LanguagePreference string  `json:"language_preference"`
	TargetRole         *string `json:"target_role"`
	PreferredZone      *string `json:"preferred_zone"`
	ReferralCode       string  `json:"referral_code"`
	ReferredByCode     *string `json:"referred_by_code"`
}

type updateWorkerProfileRequest struct {
	FullName           string  `json:"full_name"`
	LanguagePreference string  `json:"language_preference"`
	TargetRole         *string `json:"target_role"`
	PreferredZone      *string `json:"preferred_zone"`
}

func (h *WorkerHandler) Create(c *gin.Context) {
	var request createWorkerRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_json", "invalid JSON body")
		return
	}

	request.PhoneNumber = requiredString(request.PhoneNumber)
	request.FullName = requiredString(request.FullName)
	request.LanguagePreference = requiredString(request.LanguagePreference)
	request.ReferralCode = requiredString(request.ReferralCode)

	if request.PhoneNumber == "" || !phonePattern.MatchString(request.PhoneNumber) {
		writeError(c, http.StatusBadRequest, "invalid_phone_number", "phone_number is required and must be E.164 format")
		return
	}
	if request.FullName == "" {
		writeError(c, http.StatusBadRequest, "missing_full_name", "full_name is required")
		return
	}
	if request.ReferralCode == "" {
		writeError(c, http.StatusBadRequest, "missing_referral_code", "referral_code is required")
		return
	}

	worker, err := h.workers.CreateWorker(c.Request.Context(), service.CreateWorkerInput{
		PhoneNumber:        request.PhoneNumber,
		FullName:           request.FullName,
		LanguagePreference: request.LanguagePreference,
		TargetRole:         optionalString(request.TargetRole),
		PreferredZone:      optionalString(request.PreferredZone),
		ReferralCode:       request.ReferralCode,
		ReferredByCode:     optionalString(request.ReferredByCode),
	})
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"worker": newWorkerResponse(worker)})
}

func (h *WorkerHandler) GetByID(c *gin.Context) {
	worker, err := h.workers.GetWorkerByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"worker": newWorkerResponse(worker)})
}

func (h *WorkerHandler) UpdateProfile(c *gin.Context) {
	var request updateWorkerProfileRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_json", "invalid JSON body")
		return
	}

	request.FullName = requiredString(request.FullName)
	request.LanguagePreference = requiredString(request.LanguagePreference)

	if request.FullName == "" {
		writeError(c, http.StatusBadRequest, "missing_full_name", "full_name is required")
		return
	}

	worker, err := h.workers.UpdateWorkerProfile(c.Request.Context(), c.Param("id"), service.UpdateWorkerProfileInput{
		FullName:           request.FullName,
		LanguagePreference: request.LanguagePreference,
		TargetRole:         optionalString(request.TargetRole),
		PreferredZone:      optionalString(request.PreferredZone),
	})
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"worker": newWorkerResponse(worker)})
}
