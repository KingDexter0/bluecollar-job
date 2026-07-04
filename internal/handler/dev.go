package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"bluecollarjob/internal/service"

	"github.com/gin-gonic/gin"
)

type DevHandler struct {
	notifications       service.NotificationWorkerService
	notificationQueries service.NotificationQueryService
	states              service.ConversationStateService
	statusOTPs          service.StatusOTPService
	referrals           service.ReferralService
}

func NewDevHandler(notifications service.NotificationWorkerService, notificationQueries service.NotificationQueryService, states service.ConversationStateService, statusOTPs service.StatusOTPService, referrals service.ReferralService) *DevHandler {
	return &DevHandler{
		notifications:       notifications,
		notificationQueries: notificationQueries,
		states:              states,
		statusOTPs:          statusOTPs,
		referrals:           referrals,
	}
}

type devProcessNotificationsRequest struct {
	Limit int `json:"limit"`
}

type devConversationStateRequest struct {
	PhoneNumber string          `json:"phone_number"`
	State       string          `json:"state"`
	Data        json.RawMessage `json:"data"`
	TTLSeconds  int             `json:"ttl_seconds"`
}

type devGenerateStatusOTPRequest struct {
	PhoneNumber string `json:"phone_number"`
}

type devVerifyStatusOTPRequest struct {
	PhoneNumber   string `json:"phone_number"`
	TransactionID string `json:"transaction_id"`
	OTP           string `json:"otp"`
}

type devProcessReferralPayoutsRequest struct {
	Limit int `json:"limit"`
}

func (h *DevHandler) ProcessNotificationsOnce(c *gin.Context) {
	var request devProcessNotificationsRequest
	if err := c.ShouldBindJSON(&request); err != nil && !errors.Is(err, io.EOF) {
		writeError(c, http.StatusBadRequest, "invalid_json", "invalid JSON body")
		return
	}
	if request.Limit <= 0 {
		request.Limit = 10
	}

	result, err := h.notifications.ProcessOnce(c.Request.Context(), request.Limit)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"result": result})
}

func (h *DevHandler) ListNotifications(c *gin.Context) {
	limit, offset := parsePagination(c)

	notifications, err := h.notificationQueries.ListNotifications(c.Request.Context(), service.NotificationListFilters{
		Status:    c.Query("status"),
		EventType: c.Query("event_type"),
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"notifications": notifications})
}

func (h *DevHandler) SetRedisState(c *gin.Context) {
	var request devConversationStateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_json", "invalid JSON body")
		return
	}
	if !phonePattern.MatchString(requiredString(request.PhoneNumber)) {
		writeError(c, http.StatusBadRequest, "invalid_phone_number", "valid phone_number is required")
		return
	}
	if requiredString(request.State) == "" {
		writeError(c, http.StatusBadRequest, "missing_state", "state is required")
		return
	}

	ttl := time.Duration(request.TTLSeconds) * time.Second
	state, err := h.states.SetState(c.Request.Context(), request.PhoneNumber, request.State, request.Data, ttl)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"state": state})
}

func (h *DevHandler) GetRedisState(c *gin.Context) {
	state, err := h.states.GetState(c.Request.Context(), c.Param("phone"))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"state": state})
}

func (h *DevHandler) DeleteRedisState(c *gin.Context) {
	if err := h.states.DeleteState(c.Request.Context(), c.Param("phone")); err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": true})
}

func (h *DevHandler) GenerateStatusOTP(c *gin.Context) {
	var request devGenerateStatusOTPRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_json", "invalid JSON body")
		return
	}
	if !phonePattern.MatchString(requiredString(request.PhoneNumber)) {
		writeError(c, http.StatusBadRequest, "invalid_phone_number", "valid phone_number is required")
		return
	}

	result, err := h.statusOTPs.Generate(c.Request.Context(), request.PhoneNumber)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"otp": result})
}

func (h *DevHandler) VerifyStatusOTP(c *gin.Context) {
	var request devVerifyStatusOTPRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_json", "invalid JSON body")
		return
	}
	if !phonePattern.MatchString(requiredString(request.PhoneNumber)) {
		writeError(c, http.StatusBadRequest, "invalid_phone_number", "valid phone_number is required")
		return
	}
	if request.TransactionID == "" || !otpPattern.MatchString(requiredString(request.OTP)) {
		writeError(c, http.StatusBadRequest, "invalid_otp", "transaction_id and valid otp are required")
		return
	}

	if err := h.statusOTPs.Verify(c.Request.Context(), request.PhoneNumber, request.TransactionID, request.OTP); err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"verified": true})
}

func (h *DevHandler) ProcessReferralPayouts(c *gin.Context) {
	var request devProcessReferralPayoutsRequest
	if err := c.ShouldBindJSON(&request); err != nil && !errors.Is(err, io.EOF) {
		writeError(c, http.StatusBadRequest, "invalid_json", "invalid JSON body")
		return
	}
	if request.Limit <= 0 {
		request.Limit = 10
	}
	result, err := h.referrals.ProcessPendingPayouts(c.Request.Context(), request.Limit)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"result": result})
}
