package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"bluecollarjob/internal/models"
	"bluecollarjob/internal/repository"
)

func TestNotificationQueryServiceListsSafeNotifications(t *testing.T) {
	userID := "worker-1"
	processedAt := time.Now()
	repo := &queryNotificationRepository{
		events: []models.NotificationEvent{{
			ID:          "notification-1",
			UserID:      &userID,
			EventType:   "application_submitted",
			Recipient:   "+919876543210",
			Payload:     []byte(`{"job_title":"Machine Operator","aadhaar_hash":"secret-hash","otp":"123456","document_ref":"local/path.jpg"}`),
			Status:      models.NotificationStatusSent,
			LastError:   stringPointer(""),
			ProcessedAt: &processedAt,
			CreatedAt:   processedAt.Add(-time.Minute),
			UpdatedAt:   processedAt,
		}},
	}

	service := NewNotificationQueryService(repo)
	items, err := service.ListNotifications(context.Background(), NotificationListFilters{
		Status:    "Sent",
		EventType: "application_submitted",
		Limit:     25,
		Offset:    5,
	})
	if err != nil {
		t.Fatalf("list notifications: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one notification, got %d", len(items))
	}
	if repo.filters.Status == nil || *repo.filters.Status != models.NotificationStatusSent {
		t.Fatalf("expected sent status filter, got %#v", repo.filters.Status)
	}
	if repo.filters.EventType == nil || *repo.filters.EventType != "application_submitted" {
		t.Fatalf("expected event type filter, got %#v", repo.filters.EventType)
	}
	if items[0].WorkerID == nil || *items[0].WorkerID != userID {
		t.Fatalf("expected worker id to mirror user id, got %#v", items[0].WorkerID)
	}
	if items[0].MessagePreview != "Your application for Machine Operator has been submitted." {
		t.Fatalf("unexpected message preview: %q", items[0].MessagePreview)
	}

	body, err := json.Marshal(items[0])
	if err != nil {
		t.Fatalf("marshal response item: %v", err)
	}
	bodyText := string(body)
	for _, forbidden := range []string{"aadhaar", "secret-hash", "123456", "document_ref", "local/path.jpg", "payload"} {
		if strings.Contains(bodyText, forbidden) {
			t.Fatalf("response exposed sensitive field %q in %s", forbidden, bodyText)
		}
	}
}

func TestNotificationQueryServiceRejectsInvalidStatus(t *testing.T) {
	service := NewNotificationQueryService(&queryNotificationRepository{})
	_, err := service.ListNotifications(context.Background(), NotificationListFilters{Status: "Done"})
	if err == nil || !strings.Contains(err.Error(), "invalid notification status") {
		t.Fatalf("expected invalid status error, got %v", err)
	}
}

type queryNotificationRepository struct {
	events  []models.NotificationEvent
	filters repository.NotificationEventFilters
}

func (r *queryNotificationRepository) CreateNotificationEvent(ctx context.Context, event *models.NotificationEvent) (*models.NotificationEvent, error) {
	return event, nil
}

func (r *queryNotificationRepository) ClaimPendingNotificationEvents(ctx context.Context, limit int) ([]models.NotificationEvent, error) {
	return nil, nil
}

func (r *queryNotificationRepository) ListNotificationEvents(ctx context.Context, filters repository.NotificationEventFilters) ([]models.NotificationEvent, error) {
	r.filters = filters
	return r.events, nil
}

func (r *queryNotificationRepository) MarkNotificationEventSent(ctx context.Context, id string) (*models.NotificationEvent, error) {
	return &models.NotificationEvent{ID: id, Status: models.NotificationStatusSent}, nil
}

func (r *queryNotificationRepository) MarkNotificationEventFailed(ctx context.Context, id string, reason string) (*models.NotificationEvent, error) {
	return &models.NotificationEvent{ID: id, Status: models.NotificationStatusFailed, LastError: &reason}, nil
}
