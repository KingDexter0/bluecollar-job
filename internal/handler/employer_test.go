package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	appmiddleware "bluecollarjob/internal/middleware"
	"bluecollarjob/internal/models"
	"bluecollarjob/internal/service"

	"github.com/gin-gonic/gin"
)

func TestEmployerHandlerRegister(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewEmployerHandler(&fakeEmployerService{})
	router := gin.New()
	router.POST("/api/v1/employers/register", handler.Register)

	body := `{"company_name":"ACME Works","contact_name":"Anita","email":"owner@example.com","password":"secret123"}`
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/employers/register", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected %d, got %d: %s", http.StatusCreated, response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"token"`) {
		t.Fatalf("expected token in response: %s", response.Body.String())
	}
}

func TestEmployerHandlerLogin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewEmployerHandler(&fakeEmployerService{})
	router := gin.New()
	router.POST("/api/v1/employers/login", handler.Login)

	body := `{"email":"owner@example.com","password":"secret123"}`
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/employers/login", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d: %s", http.StatusOK, response.Code, response.Body.String())
	}
}

func TestEmployerAuthMiddlewareWithoutToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auth := service.NewAuthService("test-secret", "test")
	router := gin.New()
	router.GET("/protected", appmiddleware.EmployerAuth(auth), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d", http.StatusUnauthorized, response.Code)
	}
}

func TestEmployerAuthMiddlewareWithToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auth := service.NewAuthService("test-secret", "test")
	token, err := auth.GenerateEmployerToken("employer-1")
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	router := gin.New()
	router.GET("/protected", appmiddleware.EmployerAuth(auth), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"employer_id": appmiddleware.GetEmployerID(c)})
	})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d: %s", http.StatusOK, response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "employer-1") {
		t.Fatalf("expected employer id in response: %s", response.Body.String())
	}
}

type fakeEmployerService struct{}

func (s *fakeEmployerService) Register(ctx context.Context, input service.RegisterEmployerInput) (*models.Employer, string, error) {
	return fakeEmployer(input.Email), "register-token", nil
}

func (s *fakeEmployerService) Login(ctx context.Context, email, password string) (*models.Employer, string, error) {
	return fakeEmployer(email), "login-token", nil
}

func (s *fakeEmployerService) GetProfile(ctx context.Context, employerID string) (*models.Employer, error) {
	return fakeEmployer("owner@example.com"), nil
}

func (s *fakeEmployerService) UpdateProfile(ctx context.Context, employerID string, input service.UpdateEmployerProfileInput) (*models.Employer, error) {
	return fakeEmployer("owner@example.com"), nil
}

func (s *fakeEmployerService) CreateJob(ctx context.Context, employerID string, input service.EmployerJobInput) (*models.Job, error) {
	return &models.Job{ID: "job-1", EmployerID: employerID, Title: input.Title, Role: input.Role, LocationCity: input.LocationCity, LocationState: input.LocationState, ShiftSchedule: input.ShiftSchedule, CreatedAt: time.Now()}, nil
}

func (s *fakeEmployerService) ListJobs(ctx context.Context, employerID string, limit, offset int) ([]models.Job, error) {
	return nil, nil
}

func (s *fakeEmployerService) GetJob(ctx context.Context, employerID, jobID string) (*models.Job, error) {
	return nil, nil
}

func (s *fakeEmployerService) UpdateJob(ctx context.Context, employerID, jobID string, input service.EmployerJobInput) (*models.Job, error) {
	return nil, nil
}

func (s *fakeEmployerService) UpdateJobStatus(ctx context.Context, employerID, jobID string, isActive bool) (*models.Job, error) {
	return nil, nil
}

func fakeEmployer(email string) *models.Employer {
	return &models.Employer{
		ID:          "employer-1",
		CompanyName: "ACME Works",
		ContactName: "Anita",
		Email:       email,
		IsVerified:  true,
	}
}
