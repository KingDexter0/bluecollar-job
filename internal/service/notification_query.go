package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"bluecollarjob/internal/models"
	"bluecollarjob/internal/repository"
)

type NotificationQueryService interface {
	ListNotifications(ctx context.Context, filters NotificationListFilters) ([]NotificationListItem, error)
}

type NotificationListFilters struct {
	Status    string
	EventType string
	Limit     int
	Offset    int
}

type NotificationListItem struct {
	ID             string                    `json:"id"`
	UserID         *string                   `json:"user_id,omitempty"`
	WorkerID       *string                   `json:"worker_id,omitempty"`
	PhoneNumber    string                    `json:"phone_number,omitempty"`
	EventType      string                    `json:"event_type"`
	MessagePreview string                    `json:"message_preview"`
	Status         models.NotificationStatus `json:"status"`
	FailureReason  *string                   `json:"failure_reason,omitempty"`
	CreatedAt      time.Time                 `json:"created_at"`
	UpdatedAt      time.Time                 `json:"updated_at"`
	ProcessedAt    *time.Time                `json:"processed_at,omitempty"`
}

type notificationQueryService struct {
	notifications repository.NotificationRepository
}

func NewNotificationQueryService(notifications repository.NotificationRepository) NotificationQueryService {
	return &notificationQueryService{notifications: notifications}
}

func (s *notificationQueryService) ListNotifications(ctx context.Context, filters NotificationListFilters) ([]NotificationListItem, error) {
	repoFilters, err := toNotificationRepositoryFilters(filters)
	if err != nil {
		return nil, err
	}

	events, err := s.notifications.ListNotificationEvents(ctx, repoFilters)
	if err != nil {
		return nil, err
	}

	items := make([]NotificationListItem, 0, len(events))
	for _, event := range events {
		items = append(items, NotificationListItem{
			ID:             event.ID,
			UserID:         event.UserID,
			WorkerID:       event.UserID,
			PhoneNumber:    event.Recipient,
			EventType:      event.EventType,
			MessagePreview: safeNotificationPreview(event),
			Status:         event.Status,
			FailureReason:  event.LastError,
			CreatedAt:      event.CreatedAt,
			UpdatedAt:      event.UpdatedAt,
			ProcessedAt:    event.ProcessedAt,
		})
	}
	return items, nil
}

func toNotificationRepositoryFilters(filters NotificationListFilters) (repository.NotificationEventFilters, error) {
	repoFilters := repository.NotificationEventFilters{
		Limit:  filters.Limit,
		Offset: filters.Offset,
	}

	status := strings.TrimSpace(filters.Status)
	if status != "" {
		notificationStatus := models.NotificationStatus(status)
		if !validNotificationStatus(notificationStatus) {
			return repoFilters, fmt.Errorf("%w: invalid notification status", ErrInvalidInput)
		}
		repoFilters.Status = &notificationStatus
	}

	eventType := strings.TrimSpace(filters.EventType)
	if eventType != "" {
		repoFilters.EventType = &eventType
	}

	return repoFilters, nil
}

func validNotificationStatus(status models.NotificationStatus) bool {
	switch status {
	case models.NotificationStatusPending,
		models.NotificationStatusProcessing,
		models.NotificationStatusSent,
		models.NotificationStatusFailed:
		return true
	default:
		return false
	}
}

func safeNotificationPreview(event models.NotificationEvent) string {
	preview := strings.TrimSpace(buildNotificationMessage(event))
	if preview == "" {
		return "Notification update"
	}
	if len(preview) <= 160 {
		return preview
	}
	return preview[:157] + "..."
}
