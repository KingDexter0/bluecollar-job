package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"bluecollarjob/internal/models"
	"bluecollarjob/internal/repository"
)

func TestNotificationWorkerPendingBecomesSent(t *testing.T) {
	ctx := context.Background()
	repo := &workerNotificationRepository{
		pending: []models.NotificationEvent{{
			ID:        "notification-1",
			EventType: "application_submitted",
			Recipient: "+919876543210",
			Payload:   []byte(`{"job_title":"Machine Operator"}`),
			Status:    models.NotificationStatusPending,
		}},
	}
	worker := NewNotificationWorkerService(repo, NewMockWhatsAppSender(), NotificationWorkerConfig{BatchSize: 10})

	result, err := worker.ProcessOnce(ctx, 10)
	if err != nil {
		t.Fatalf("process once: %v", err)
	}
	if result.Sent != 1 || result.Failed != 0 {
		t.Fatalf("expected one sent, got %#v", result)
	}
	if repo.sentID != "notification-1" {
		t.Fatalf("expected notification marked sent, got %s", repo.sentID)
	}
}

func TestNotificationWorkerRetriesTemporaryFailure(t *testing.T) {
	ctx := context.Background()
	repo := &workerNotificationRepository{
		pending: []models.NotificationEvent{{
			ID:        "notification-1",
			EventType: "application_shortlisted",
			Recipient: "+919876543210",
			Payload:   []byte(`{"job_title":"Machine Operator"}`),
			Status:    models.NotificationStatusPending,
		}},
	}
	sender := &flakyWhatsAppSender{temporaryFailures: 1}
	worker := NewNotificationWorkerService(repo, sender, NotificationWorkerConfig{
		BatchSize:    10,
		MaxAttempts:  2,
		RetryBackoff: time.Millisecond,
	})

	result, err := worker.ProcessOnce(ctx, 10)
	if err != nil {
		t.Fatalf("process once: %v", err)
	}
	if result.Sent != 1 || sender.calls != 2 {
		t.Fatalf("expected retry then sent, result=%#v calls=%d", result, sender.calls)
	}
}

func TestNotificationWorkerFailureBecomesFailed(t *testing.T) {
	ctx := context.Background()
	repo := &workerNotificationRepository{
		pending: []models.NotificationEvent{{
			ID:        "notification-1",
			EventType: "interview_scheduled",
			Recipient: "+919876543210",
			Payload:   []byte(`{"job_title":"Machine Operator"}`),
			Status:    models.NotificationStatusPending,
		}},
	}
	sender := NewMockWhatsAppSender()
	sender.FailRecipients["+919876543210"] = true
	worker := NewNotificationWorkerService(repo, sender, NotificationWorkerConfig{BatchSize: 10})

	result, err := worker.ProcessOnce(ctx, 10)
	if err != nil {
		t.Fatalf("process once: %v", err)
	}
	if result.Failed != 1 || result.Sent != 0 {
		t.Fatalf("expected one failed, got %#v", result)
	}
	if repo.failedID != "notification-1" || repo.failedReason == "" {
		t.Fatalf("expected failed notification reason, got id=%s reason=%q", repo.failedID, repo.failedReason)
	}
}

type workerNotificationRepository struct {
	pending      []models.NotificationEvent
	sentID       string
	failedID     string
	failedReason string
	err          error
}

func (r *workerNotificationRepository) CreateNotificationEvent(ctx context.Context, event *models.NotificationEvent) (*models.NotificationEvent, error) {
	return event, nil
}

func (r *workerNotificationRepository) ClaimPendingNotificationEvents(ctx context.Context, limit int) ([]models.NotificationEvent, error) {
	if r.err != nil {
		return nil, r.err
	}
	if limit <= 0 || limit > len(r.pending) {
		limit = len(r.pending)
	}
	events := append([]models.NotificationEvent(nil), r.pending[:limit]...)
	r.pending = r.pending[limit:]
	for i := range events {
		events[i].Status = models.NotificationStatusProcessing
	}
	return events, nil
}

func (r *workerNotificationRepository) ListNotificationEvents(ctx context.Context, filters repository.NotificationEventFilters) ([]models.NotificationEvent, error) {
	return nil, nil
}

func (r *workerNotificationRepository) MarkNotificationEventSent(ctx context.Context, id string) (*models.NotificationEvent, error) {
	if id == "" {
		return nil, errors.New("missing id")
	}
	r.sentID = id
	return &models.NotificationEvent{ID: id, Status: models.NotificationStatusSent}, nil
}

func (r *workerNotificationRepository) MarkNotificationEventFailed(ctx context.Context, id string, reason string) (*models.NotificationEvent, error) {
	if id == "" {
		return nil, errors.New("missing id")
	}
	r.failedID = id
	r.failedReason = reason
	return &models.NotificationEvent{ID: id, Status: models.NotificationStatusFailed, LastError: &reason}, nil
}

type flakyWhatsAppSender struct {
	calls             int
	temporaryFailures int
}

func (s *flakyWhatsAppSender) SendMessage(ctx context.Context, phoneNumber, message string) error {
	s.calls++
	if s.calls <= s.temporaryFailures {
		return fmt.Errorf("%w: timeout", ErrTemporaryWhatsAppDelivery)
	}
	return nil
}
