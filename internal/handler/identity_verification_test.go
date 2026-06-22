package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bluecollarjob/internal/models"

	"github.com/gin-gonic/gin"
)

func TestIdentityVerificationHandlerRejectsInvalidAadhaar(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &fakeIdentityVerificationService{}
	handler := NewIdentityVerificationHandler(service)
	router := gin.New()
	router.POST("/api/v1/workers/:id/identity/aadhaar/start", handler.StartAadhaar)

	body := `{"aadhaar_number":"12345","consent_given":true}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/workers/user-1/identity/aadhaar/start", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
	if service.startCalled {
		t.Fatal("service should not be called for invalid aadhaar")
	}
}

type fakeIdentityVerificationService struct {
	startCalled bool
}

func (s *fakeIdentityVerificationService) StartAadhaarOTP(ctx context.Context, userID, aadhaarNumber string, consentGiven bool) (*models.WorkerIdentityVerification, error) {
	s.startCalled = true
	return nil, nil
}

func (s *fakeIdentityVerificationService) VerifyAadhaarOTP(ctx context.Context, userID, transactionID, otp string) (*models.WorkerIdentityVerification, error) {
	return nil, nil
}

func (s *fakeIdentityVerificationService) MarkDocumentUploaded(ctx context.Context, userID, documentRef string) (*models.WorkerIdentityVerification, error) {
	return nil, nil
}

func (s *fakeIdentityVerificationService) MarkSkipped(ctx context.Context, userID, reason string) (*models.WorkerIdentityVerification, error) {
	return nil, nil
}

func (s *fakeIdentityVerificationService) GetLatest(ctx context.Context, userID string) (*models.WorkerIdentityVerification, error) {
	return nil, nil
}
