package handler

import (
	"net/http"

	"bluecollarjob/internal/service"

	"github.com/gin-gonic/gin"
)

type IdentityVerificationHandler struct {
	verifications service.IdentityVerificationService
}

func NewIdentityVerificationHandler(verifications service.IdentityVerificationService) *IdentityVerificationHandler {
	return &IdentityVerificationHandler{verifications: verifications}
}

type startAadhaarRequest struct {
	AadhaarNumber string `json:"aadhaar_number"`
	ConsentGiven  bool   `json:"consent_given"`
}

type verifyAadhaarRequest struct {
	TransactionID string `json:"transaction_id"`
	OTP           string `json:"otp"`
}

type documentVerificationRequest struct {
	DocumentRef string `json:"document_ref"`
}

type skipVerificationRequest struct {
	Reason string `json:"reason"`
}

func (h *IdentityVerificationHandler) StartAadhaar(c *gin.Context) {
	var request startAadhaarRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_json", "invalid JSON body")
		return
	}

	request.AadhaarNumber = requiredString(request.AadhaarNumber)
	if !aadhaarPattern.MatchString(request.AadhaarNumber) {
		writeError(c, http.StatusBadRequest, "invalid_aadhaar", "aadhaar_number must contain exactly 12 digits")
		return
	}
	if !request.ConsentGiven {
		writeError(c, http.StatusBadRequest, "missing_consent", "consent_given must be true")
		return
	}

	verification, err := h.verifications.StartAadhaarOTP(c.Request.Context(), c.Param("id"), request.AadhaarNumber, request.ConsentGiven)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"identity_verification": newIdentityVerificationResponse(verification)})
}

func (h *IdentityVerificationHandler) VerifyAadhaar(c *gin.Context) {
	var request verifyAadhaarRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_json", "invalid JSON body")
		return
	}

	request.TransactionID = requiredString(request.TransactionID)
	request.OTP = requiredString(request.OTP)
	if request.TransactionID == "" {
		writeError(c, http.StatusBadRequest, "missing_transaction_id", "transaction_id is required")
		return
	}
	if !otpPattern.MatchString(request.OTP) {
		writeError(c, http.StatusBadRequest, "invalid_otp", "otp is required and must be 4 to 8 digits")
		return
	}

	verification, err := h.verifications.VerifyAadhaarOTP(c.Request.Context(), c.Param("id"), request.TransactionID, request.OTP)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"identity_verification": newIdentityVerificationResponse(verification)})
}

func (h *IdentityVerificationHandler) MarkDocumentUploaded(c *gin.Context) {
	var request documentVerificationRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_json", "invalid JSON body")
		return
	}

	request.DocumentRef = requiredString(request.DocumentRef)
	if request.DocumentRef == "" {
		writeError(c, http.StatusBadRequest, "missing_document_ref", "document_ref cannot be empty")
		return
	}

	verification, err := h.verifications.MarkDocumentUploaded(c.Request.Context(), c.Param("id"), request.DocumentRef)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"identity_verification": newIdentityVerificationResponse(verification)})
}

func (h *IdentityVerificationHandler) MarkSkipped(c *gin.Context) {
	var request skipVerificationRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_json", "invalid JSON body")
		return
	}

	verification, err := h.verifications.MarkSkipped(c.Request.Context(), c.Param("id"), requiredString(request.Reason))
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"identity_verification": newIdentityVerificationResponse(verification)})
}

func (h *IdentityVerificationHandler) GetLatest(c *gin.Context) {
	verification, err := h.verifications.GetLatest(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"identity_verification": newIdentityVerificationResponse(verification)})
}
