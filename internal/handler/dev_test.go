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

func TestDevHandlerListNotifications(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Now().UTC()
	queryService := &fakeNotificationQueryService{
		items: []service.NotificationListItem{{
			ID:             "notification-1",
			UserID:         testStringPointer("worker-1"),
			WorkerID:       testStringPointer("worker-1"),
			PhoneNumber:    "+919876543210",
			EventType:      "application_submitted",
			MessagePreview: "Your application for Machine Operator has been submitted.",
			Status:         models.NotificationStatusPending,
			CreatedAt:      now,
			UpdatedAt:      now,
		}},
	}
	handler := NewDevHandler(nil, queryService, nil, nil, nil)
	router := gin.New()
	router.GET("/api/v1/dev/notifications", handler.ListNotifications)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/dev/notifications?status=Pending&event_type=application_submitted&limit=10&offset=5", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, response.Code, response.Body.String())
	}
	if queryService.filters.Status != "Pending" || queryService.filters.EventType != "application_submitted" || queryService.filters.Limit != 10 || queryService.filters.Offset != 5 {
		t.Fatalf("unexpected filters: %#v", queryService.filters)
	}
	body := response.Body.String()
	for _, expected := range []string{`"id":"notification-1"`, `"worker_id":"worker-1"`, `"phone_number":"+919876543210"`, `"message_preview":"Your application for Machine Operator has been submitted."`} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected response to contain %s, got %s", expected, body)
		}
	}
	for _, forbidden := range []string{"aadhaar", "otp", "password_hash", "aadhaar_hash", "document_ref", "payload"} {
		if strings.Contains(strings.ToLower(body), forbidden) {
			t.Fatalf("response exposed sensitive field %q: %s", forbidden, body)
		}
	}
}

type fakeNotificationQueryService struct {
	filters service.NotificationListFilters
	items   []service.NotificationListItem
}

func (s *fakeNotificationQueryService) ListNotifications(ctx context.Context, filters service.NotificationListFilters) ([]service.NotificationListItem, error) {
	s.filters = filters
	return s.items, nil
}

func testStringPointer(value string) *string {
	return &value
}
