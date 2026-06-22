package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bluecollarjob/internal/models"
	"bluecollarjob/internal/service"

	"github.com/gin-gonic/gin"
)

func TestWorkerHandlerCreate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewWorkerHandler(&fakeWorkerService{})
	router := gin.New()
	router.POST("/api/v1/workers", handler.Create)

	body := `{"phone_number":"+919876543299","full_name":"Test Worker","language_preference":"en","referral_code":"TEST299"}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/workers", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"verification_tier":"High"`) {
		t.Fatalf("expected high verification tier response, got %s", response.Body.String())
	}
}

type fakeWorkerService struct{}

func (s *fakeWorkerService) CreateWorker(ctx context.Context, input service.CreateWorkerInput) (*models.User, error) {
	return &models.User{
		ID:                 "worker-test-id",
		PhoneNumber:        input.PhoneNumber,
		FullName:           input.FullName,
		LanguagePreference: input.LanguagePreference,
		VerificationTier:   models.VerificationTierHigh,
		ReferralCode:       input.ReferralCode,
		IsActive:           true,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}, nil
}

func (s *fakeWorkerService) GetWorkerByID(ctx context.Context, id string) (*models.User, error) {
	return nil, nil
}

func (s *fakeWorkerService) GetWorkerByPhone(ctx context.Context, phoneNumber string) (*models.User, error) {
	return nil, nil
}

func (s *fakeWorkerService) UpdateWorkerProfile(ctx context.Context, id string, input service.UpdateWorkerProfileInput) (*models.User, error) {
	return nil, nil
}

func (s *fakeWorkerService) GetWorkerByReferralCode(ctx context.Context, referralCode string) (*models.User, error) {
	return nil, nil
}
