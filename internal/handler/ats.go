package handler

import (
	"net/http"
	"time"

	appmiddleware "bluecollarjob/internal/middleware"
	"bluecollarjob/internal/models"
	"bluecollarjob/internal/repository"
	"bluecollarjob/internal/service"

	"github.com/gin-gonic/gin"
)

type ATSHandler struct {
	ats service.ATSService
}

func NewATSHandler(ats service.ATSService) *ATSHandler {
	return &ATSHandler{ats: ats}
}

type updateApplicationStatusRequest struct {
	Status models.ApplicationStatus `json:"status"`
}

type interviewScheduleRequest struct {
	StartsAt        string  `json:"starts_at"`
	EndsAt          string  `json:"ends_at"`
	Timezone        string  `json:"timezone"`
	FactoryLocation *string `json:"factory_location"`
	GoogleMapsURL   *string `json:"google_maps_url"`
}

type createInterviewSlotsRequest struct {
	Slots []interviewScheduleRequest `json:"slots"`
}

type selectInterviewSlotRequest struct {
	SlotID string `json:"slot_id"`
}

func (h *ATSHandler) ListEmployerApplications(c *gin.Context) {
	filters, ok := bindEmployerApplicationFilters(c)
	if !ok {
		return
	}

	applications, err := h.ats.ListEmployerApplications(c.Request.Context(), appmiddleware.GetEmployerID(c), filters)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	response := make([]applicationATSResponse, 0, len(applications))
	for _, application := range applications {
		response = append(response, newApplicationATSResponse(application))
	}
	c.JSON(http.StatusOK, gin.H{"applications": response})
}

func (h *ATSHandler) ListJobApplications(c *gin.Context) {
	filters, ok := bindEmployerApplicationFilters(c)
	if !ok {
		return
	}
	jobID := requiredString(c.Param("job_id"))
	if jobID == "" {
		jobID = requiredString(c.Param("id"))
	}
	filters.JobID = &jobID

	applications, err := h.ats.ListEmployerApplications(c.Request.Context(), appmiddleware.GetEmployerID(c), filters)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	response := make([]applicationATSResponse, 0, len(applications))
	for _, application := range applications {
		response = append(response, newApplicationATSResponse(application))
	}
	c.JSON(http.StatusOK, gin.H{"applications": response})
}

func (h *ATSHandler) GetEmployerApplication(c *gin.Context) {
	application, err := h.ats.GetEmployerApplication(c.Request.Context(), appmiddleware.GetEmployerID(c), c.Param("id"))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"application": newApplicationATSResponse(*application)})
}

func (h *ATSHandler) UpdateApplicationStatus(c *gin.Context) {
	var request updateApplicationStatusRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_json", "invalid JSON body")
		return
	}
	if !validApplicationStatus(request.Status) {
		writeError(c, http.StatusBadRequest, "invalid_status", "valid application status is required")
		return
	}

	application, err := h.ats.UpdateApplicationStatus(c.Request.Context(), appmiddleware.GetEmployerID(c), c.Param("id"), request.Status)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"application": newApplicationATSResponse(*application)})
}

func (h *ATSHandler) ScheduleDirectInterview(c *gin.Context) {
	var request interviewScheduleRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_json", "invalid JSON body")
		return
	}

	input, ok := bindInterviewSchedule(c, request)
	if !ok {
		return
	}

	slot, application, err := h.ats.ScheduleDirectInterview(c.Request.Context(), appmiddleware.GetEmployerID(c), c.Param("id"), input)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"application":    newApplicationATSResponse(*application),
		"interview_slot": newInterviewSlotResponse(*slot),
	})
}

func (h *ATSHandler) CreateInterviewSlots(c *gin.Context) {
	var request createInterviewSlotsRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_json", "invalid JSON body")
		return
	}
	if len(request.Slots) != 3 {
		writeError(c, http.StatusBadRequest, "invalid_slots", "exactly 3 interview slots are required")
		return
	}

	slots := make([]service.InterviewScheduleInput, 0, len(request.Slots))
	for _, slotRequest := range request.Slots {
		slot, ok := bindInterviewSchedule(c, slotRequest)
		if !ok {
			return
		}
		slots = append(slots, slot)
	}

	createdSlots, application, err := h.ats.CreateInterviewSlots(c.Request.Context(), appmiddleware.GetEmployerID(c), c.Param("id"), slots)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	response := make([]interviewSlotResponse, 0, len(createdSlots))
	for _, slot := range createdSlots {
		response = append(response, newInterviewSlotResponse(slot))
	}
	c.JSON(http.StatusCreated, gin.H{
		"application":     newApplicationATSResponse(*application),
		"interview_slots": response,
	})
}

func (h *ATSHandler) SelectInterviewSlot(c *gin.Context) {
	var request selectInterviewSlotRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_json", "invalid JSON body")
		return
	}
	request.SlotID = requiredString(request.SlotID)
	if request.SlotID == "" {
		writeError(c, http.StatusBadRequest, "missing_slot_id", "slot_id is required")
		return
	}

	slot, application, err := h.ats.SelectInterviewSlot(c.Request.Context(), c.Param("id"), request.SlotID)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"application":    newApplicationResponse(*application),
		"interview_slot": newInterviewSlotResponse(*slot),
	})
}

func bindEmployerApplicationFilters(c *gin.Context) (repository.EmployerApplicationFilters, bool) {
	limit, offset := parsePagination(c)
	filters := repository.EmployerApplicationFilters{
		Limit:  limit,
		Offset: offset,
	}

	if jobID := requiredString(c.Query("job_id")); jobID != "" {
		filters.JobID = &jobID
	}
	if statusValue := requiredString(c.Query("status")); statusValue != "" {
		status := models.ApplicationStatus(statusValue)
		if !validApplicationStatus(status) {
			writeError(c, http.StatusBadRequest, "invalid_status", "valid application status is required")
			return filters, false
		}
		filters.Status = &status
	}
	if tierValue := requiredString(c.Query("verification_tier")); tierValue != "" {
		tier := models.VerificationTier(tierValue)
		if !validVerificationTier(tier) {
			writeError(c, http.StatusBadRequest, "invalid_verification_tier", "verification_tier must be Low, Medium, or High")
			return filters, false
		}
		filters.VerificationTier = &tier
	}
	if targetRole := requiredString(c.Query("target_role")); targetRole != "" {
		filters.TargetRole = &targetRole
	}
	if preferredZone := requiredString(c.Query("preferred_zone")); preferredZone != "" {
		filters.PreferredZone = &preferredZone
	}
	return filters, true
}

func bindInterviewSchedule(c *gin.Context, request interviewScheduleRequest) (service.InterviewScheduleInput, bool) {
	startsAt, ok := parseRequiredRFC3339(c, request.StartsAt, "starts_at")
	if !ok {
		return service.InterviewScheduleInput{}, false
	}
	endsAt, ok := parseRequiredRFC3339(c, request.EndsAt, "ends_at")
	if !ok {
		return service.InterviewScheduleInput{}, false
	}
	factoryLocation := optionalString(request.FactoryLocation)
	if factoryLocation == nil {
		writeError(c, http.StatusBadRequest, "missing_factory_location", "factory_location is required")
		return service.InterviewScheduleInput{}, false
	}
	googleMapsURL := optionalString(request.GoogleMapsURL)
	if googleMapsURL == nil {
		writeError(c, http.StatusBadRequest, "missing_google_maps_url", "google_maps_url is required")
		return service.InterviewScheduleInput{}, false
	}

	return service.InterviewScheduleInput{
		StartsAt:        startsAt,
		EndsAt:          endsAt,
		Timezone:        requiredString(request.Timezone),
		FactoryLocation: factoryLocation,
		GoogleMapsURL:   googleMapsURL,
	}, true
}

func parseRequiredRFC3339(c *gin.Context, value, field string) (time.Time, bool) {
	value = requiredString(value)
	if value == "" {
		writeError(c, http.StatusBadRequest, "missing_"+field, field+" is required")
		return time.Time{}, false
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid_"+field, field+" must be RFC3339")
		return time.Time{}, false
	}
	return parsed, true
}

func validApplicationStatus(status models.ApplicationStatus) bool {
	switch status {
	case models.ApplicationStatusApplied,
		models.ApplicationStatusShortlisted,
		models.ApplicationStatusSlotSelectionPending,
		models.ApplicationStatusInterviewScheduled,
		models.ApplicationStatusSelected,
		models.ApplicationStatusRejected:
		return true
	default:
		return false
	}
}

func validVerificationTier(tier models.VerificationTier) bool {
	switch tier {
	case models.VerificationTierLow, models.VerificationTierMedium, models.VerificationTierHigh:
		return true
	default:
		return false
	}
}
