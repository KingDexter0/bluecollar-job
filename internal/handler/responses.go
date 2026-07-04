package handler

import (
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"bluecollarjob/internal/middleware"
	"bluecollarjob/internal/models"
	"bluecollarjob/internal/repository"
	"bluecollarjob/internal/service"

	"github.com/gin-gonic/gin"
)

var (
	phonePattern   = regexp.MustCompile(`^\+[1-9][0-9]{7,14}$`)
	aadhaarPattern = regexp.MustCompile(`^[0-9]{12}$`)
	otpPattern     = regexp.MustCompile(`^[0-9]{4,8}$`)
	emailPattern   = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
)

func writeError(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, middleware.ErrorResponse{
		Error: middleware.ErrorBody{
			Code:    code,
			Message: message,
		},
	})
}

func writeServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, repository.ErrNotFound), errors.Is(err, service.ErrNotFound):
		writeError(c, http.StatusNotFound, "not_found", "resource not found")
	case errors.Is(err, service.ErrInvalidInput):
		writeError(c, http.StatusBadRequest, "invalid_request", err.Error())
	case errors.Is(err, repository.ErrConflict), errors.Is(err, service.ErrConflict):
		writeError(c, http.StatusConflict, "conflict", err.Error())
	default:
		writeError(c, http.StatusInternalServerError, "internal_server_error", "internal server error")
	}
}

func parsePagination(c *gin.Context) (int, int) {
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if err != nil || limit <= 0 || limit > 100 {
		limit = 50
	}

	offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if err != nil || offset < 0 {
		offset = 0
	}

	return limit, offset
}

func requiredString(value string) string {
	return strings.TrimSpace(value)
}

func optionalString(value *string) *string {
	if value == nil {
		return nil
	}
	cleaned := strings.TrimSpace(*value)
	if cleaned == "" {
		return nil
	}
	return &cleaned
}

type workerResponse struct {
	ID                 string                  `json:"id"`
	PhoneNumber        string                  `json:"phone_number"`
	FullName           string                  `json:"full_name"`
	LanguagePreference string                  `json:"language_preference"`
	TargetRole         *string                 `json:"target_role,omitempty"`
	PreferredZone      *string                 `json:"preferred_zone,omitempty"`
	VerificationTier   models.VerificationTier `json:"verification_tier"`
	ReferralCode       string                  `json:"referral_code"`
	ReferredByCode     *string                 `json:"referred_by_code,omitempty"`
	IsActive           bool                    `json:"is_active"`
}

func newWorkerResponse(user *models.User) workerResponse {
	return workerResponse{
		ID:                 user.ID,
		PhoneNumber:        user.PhoneNumber,
		FullName:           user.FullName,
		LanguagePreference: user.LanguagePreference,
		TargetRole:         user.TargetRole,
		PreferredZone:      user.PreferredZone,
		VerificationTier:   user.VerificationTier,
		ReferralCode:       user.ReferralCode,
		ReferredByCode:     user.ReferredByCode,
		IsActive:           user.IsActive,
	}
}

type identityVerificationResponse struct {
	ID                  string                            `json:"id"`
	UserID              string                            `json:"user_id"`
	Method              models.IdentityVerificationMethod `json:"method"`
	Status              models.IdentityVerificationStatus `json:"status"`
	AadhaarLast4        *string                           `json:"aadhaar_last4,omitempty"`
	AadhaarMasked       *string                           `json:"aadhaar_masked,omitempty"`
	AadhaarReferenceKey *string                           `json:"aadhaar_reference_key,omitempty"`
	DocumentUploaded    bool                              `json:"document_uploaded"`
	ConsentGiven        bool                              `json:"consent_given"`
	ConsentGivenAt      any                               `json:"consent_given_at,omitempty"`
	VerifiedAt          any                               `json:"verified_at,omitempty"`
	FailedReason        *string                           `json:"failed_reason,omitempty"`
}

func newIdentityVerificationResponse(verification *models.WorkerIdentityVerification) identityVerificationResponse {
	return identityVerificationResponse{
		ID:                  verification.ID,
		UserID:              verification.UserID,
		Method:              verification.Method,
		Status:              verification.Status,
		AadhaarLast4:        verification.AadhaarLast4,
		AadhaarMasked:       verification.AadhaarMasked,
		AadhaarReferenceKey: verification.AadhaarReferenceKey,
		DocumentUploaded:    verification.DocumentRef != nil,
		ConsentGiven:        verification.ConsentGiven,
		ConsentGivenAt:      verification.ConsentGivenAt,
		VerifiedAt:          verification.VerifiedAt,
		FailedReason:        verification.FailedReason,
	}
}

type jobResponse struct {
	ID                       string                  `json:"id"`
	EmployerID               string                  `json:"employer_id"`
	Title                    string                  `json:"title"`
	Role                     string                  `json:"role"`
	Description              string                  `json:"description"`
	SkillCategory            string                  `json:"skill_category"`
	LocationCity             string                  `json:"location_city"`
	LocationState            string                  `json:"location_state"`
	ShiftSchedule            string                  `json:"shift_schedule"`
	WageMinPaise             *int                    `json:"wage_min_paise,omitempty"`
	WageMaxPaise             *int                    `json:"wage_max_paise,omitempty"`
	RequiredVerificationTier models.VerificationTier `json:"required_verification_tier"`
	Openings                 int                     `json:"openings"`
	IsActive                 bool                    `json:"is_active"`
}

func newJobResponse(job models.Job) jobResponse {
	return jobResponse{
		ID:                       job.ID,
		EmployerID:               job.EmployerID,
		Title:                    job.Title,
		Role:                     job.Role,
		Description:              job.Description,
		SkillCategory:            job.SkillCategory,
		LocationCity:             job.LocationCity,
		LocationState:            job.LocationState,
		ShiftSchedule:            job.ShiftSchedule,
		WageMinPaise:             job.WageMinPaise,
		WageMaxPaise:             job.WageMaxPaise,
		RequiredVerificationTier: job.RequiredVerificationTier,
		Openings:                 job.Openings,
		IsActive:                 job.IsActive,
	}
}

type employerResponse struct {
	ID          string  `json:"id"`
	CompanyName string  `json:"company_name"`
	ContactName string  `json:"contact_name"`
	Email       string  `json:"email"`
	PhoneNumber *string `json:"phone_number,omitempty"`
	City        *string `json:"city,omitempty"`
	State       *string `json:"state,omitempty"`
	IsVerified  bool    `json:"is_verified"`
}

func newEmployerResponse(employer *models.Employer) employerResponse {
	return employerResponse{
		ID:          employer.ID,
		CompanyName: employer.CompanyName,
		ContactName: employer.ContactName,
		Email:       employer.Email,
		PhoneNumber: employer.PhoneNumber,
		City:        employer.City,
		State:       employer.State,
		IsVerified:  employer.IsVerified,
	}
}

type applicationResponse struct {
	ID         string                   `json:"id"`
	UserID     string                   `json:"user_id"`
	JobID      string                   `json:"job_id"`
	EmployerID string                   `json:"employer_id"`
	Status     models.ApplicationStatus `json:"status"`
	Source     string                   `json:"source"`
	AppliedAt  any                      `json:"applied_at"`
}

func newApplicationResponse(application models.Application) applicationResponse {
	return applicationResponse{
		ID:         application.ID,
		UserID:     application.UserID,
		JobID:      application.JobID,
		EmployerID: application.EmployerID,
		Status:     application.Status,
		Source:     application.Source,
		AppliedAt:  application.AppliedAt,
	}
}

type applicationATSResponse struct {
	ID                     string                   `json:"id"`
	UserID                 string                   `json:"user_id"`
	JobID                  string                   `json:"job_id"`
	EmployerID             string                   `json:"employer_id"`
	Status                 models.ApplicationStatus `json:"status"`
	Source                 string                   `json:"source"`
	AppliedAt              any                      `json:"applied_at"`
	WorkerFullName         string                   `json:"worker_full_name"`
	WorkerPhoneNumber      string                   `json:"worker_phone_number"`
	WorkerVerificationTier models.VerificationTier  `json:"worker_verification_tier"`
	WorkerTargetRole       *string                  `json:"worker_target_role,omitempty"`
	WorkerPreferredZone    *string                  `json:"worker_preferred_zone,omitempty"`
	JobTitle               string                   `json:"job_title"`
	JobRole                string                   `json:"job_role"`
}

func newApplicationATSResponse(application models.ApplicationATS) applicationATSResponse {
	return applicationATSResponse{
		ID:                     application.ID,
		UserID:                 application.UserID,
		JobID:                  application.JobID,
		EmployerID:             application.EmployerID,
		Status:                 application.Status,
		Source:                 application.Source,
		AppliedAt:              application.AppliedAt,
		WorkerFullName:         application.WorkerFullName,
		WorkerPhoneNumber:      application.WorkerPhoneNumber,
		WorkerVerificationTier: application.WorkerVerificationTier,
		WorkerTargetRole:       application.WorkerTargetRole,
		WorkerPreferredZone:    application.WorkerPreferredZone,
		JobTitle:               application.JobTitle,
		JobRole:                application.JobRole,
	}
}

type interviewSlotResponse struct {
	ID              string                     `json:"id"`
	ApplicationID   string                     `json:"application_id"`
	StartsAt        any                        `json:"starts_at"`
	EndsAt          any                        `json:"ends_at"`
	Timezone        string                     `json:"timezone"`
	FactoryLocation *string                    `json:"factory_location,omitempty"`
	GoogleMapsURL   *string                    `json:"google_maps_url,omitempty"`
	Status          models.InterviewSlotStatus `json:"status"`
	LockedUntil     any                        `json:"locked_until,omitempty"`
	ConfirmedAt     any                        `json:"confirmed_at,omitempty"`
}

func newInterviewSlotResponse(slot models.InterviewSlot) interviewSlotResponse {
	return interviewSlotResponse{
		ID:              slot.ID,
		ApplicationID:   slot.ApplicationID,
		StartsAt:        slot.StartsAt,
		EndsAt:          slot.EndsAt,
		Timezone:        slot.Timezone,
		FactoryLocation: slot.FactoryLocation,
		GoogleMapsURL:   slot.GoogleMapsURL,
		Status:          slot.Status,
		LockedUntil:     slot.LockedUntil,
		ConfirmedAt:     slot.ConfirmedAt,
	}
}

type referralResponse struct {
	ID             string  `json:"id"`
	ReferrerUserID string  `json:"referrer_user_id"`
	ReferredUserID *string `json:"referred_user_id,omitempty"`
	ReferralCode   string  `json:"referral_code"`
	CreatedAt      any     `json:"created_at"`
	ConvertedAt    any     `json:"converted_at,omitempty"`
}

func newReferralResponse(referral models.Referral) referralResponse {
	return referralResponse{
		ID:             referral.ID,
		ReferrerUserID: referral.ReferrerUserID,
		ReferredUserID: referral.ReferredUserID,
		ReferralCode:   referral.ReferralCode,
		CreatedAt:      referral.CreatedAt,
		ConvertedAt:    referral.ConvertedAt,
	}
}

type referralTransactionResponse struct {
	ID                string  `json:"id"`
	ReferralID        string  `json:"referral_id"`
	UserID            string  `json:"user_id"`
	AmountPaise       int     `json:"amount_paise"`
	Currency          string  `json:"currency"`
	Status            string  `json:"status"`
	ExternalReference *string `json:"external_reference,omitempty"`
	CreatedAt         any     `json:"created_at"`
	PaidAt            any     `json:"paid_at,omitempty"`
}

func newReferralTransactionResponse(transaction models.ReferralTransaction) referralTransactionResponse {
	return referralTransactionResponse{
		ID:                transaction.ID,
		ReferralID:        transaction.ReferralID,
		UserID:            transaction.UserID,
		AmountPaise:       transaction.AmountPaise,
		Currency:          transaction.Currency,
		Status:            transaction.Status,
		ExternalReference: transaction.ExternalReference,
		CreatedAt:         transaction.CreatedAt,
		PaidAt:            transaction.PaidAt,
	}
}
