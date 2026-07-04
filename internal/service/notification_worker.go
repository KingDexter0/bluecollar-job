package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"bluecollarjob/internal/models"
	"bluecollarjob/internal/repository"
)

type WhatsAppSender interface {
	SendMessage(ctx context.Context, phoneNumber, message string) error
}

type NotificationWorkerService interface {
	ProcessOnce(ctx context.Context, limit int) (NotificationProcessResult, error)
	Start(ctx context.Context)
}

type NotificationWorkerConfig struct {
	WorkerCount  int
	PollInterval time.Duration
	BatchSize    int
}

type NotificationProcessResult struct {
	Claimed int `json:"claimed"`
	Sent    int `json:"sent"`
	Failed  int `json:"failed"`
}

type notificationWorkerService struct {
	notifications repository.NotificationRepository
	sender        WhatsAppSender
	cfg           NotificationWorkerConfig
}

func NewNotificationWorkerService(notifications repository.NotificationRepository, sender WhatsAppSender, cfg NotificationWorkerConfig) NotificationWorkerService {
	if cfg.WorkerCount <= 0 {
		cfg.WorkerCount = 1
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 5 * time.Second
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 10
	}
	return &notificationWorkerService{
		notifications: notifications,
		sender:        sender,
		cfg:           cfg,
	}
}

func (s *notificationWorkerService) Start(ctx context.Context) {
	for i := 0; i < s.cfg.WorkerCount; i++ {
		go func() {
			ticker := time.NewTicker(s.cfg.PollInterval)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					_, _ = s.ProcessOnce(ctx, s.cfg.BatchSize)
				}

				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
				}
			}
		}()
	}
}

func (s *notificationWorkerService) ProcessOnce(ctx context.Context, limit int) (NotificationProcessResult, error) {
	events, err := s.notifications.ClaimPendingNotificationEvents(ctx, limit)
	if err != nil {
		return NotificationProcessResult{}, err
	}

	result := NotificationProcessResult{Claimed: len(events)}
	for _, event := range events {
		message := buildNotificationMessage(event)
		if err := s.sender.SendMessage(ctx, event.Recipient, message); err != nil {
			result.Failed++
			if _, markErr := s.notifications.MarkNotificationEventFailed(ctx, event.ID, err.Error()); markErr != nil {
				return result, markErr
			}
			continue
		}
		result.Sent++
		if _, err := s.notifications.MarkNotificationEventSent(ctx, event.ID); err != nil {
			return result, err
		}
	}

	return result, nil
}

type MockWhatsAppSender struct {
	FailRecipients map[string]bool
}

func NewMockWhatsAppSender() *MockWhatsAppSender {
	return &MockWhatsAppSender{FailRecipients: map[string]bool{}}
}

func (s *MockWhatsAppSender) SendMessage(ctx context.Context, phoneNumber, message string) error {
	if strings.TrimSpace(phoneNumber) == "" {
		return fmt.Errorf("recipient phone number is required")
	}
	if s.FailRecipients != nil && s.FailRecipients[phoneNumber] {
		return fmt.Errorf("mock WhatsApp delivery failed")
	}
	return nil
}

func buildNotificationMessage(event models.NotificationEvent) string {
	payload := map[string]any{}
	_ = json.Unmarshal(event.Payload, &payload)

	jobTitle := stringFromPayload(payload, "job_title", "the job")
	switch event.EventType {
	case "application_submitted":
		return fmt.Sprintf("Your application for %s has been submitted.", jobTitle)
	case "application_shortlisted":
		return fmt.Sprintf("Good news. You have been shortlisted for %s.", jobTitle)
	case "interview_slot_selection_pending":
		return fmt.Sprintf("Please select an interview slot for %s.", jobTitle)
	case "interview_scheduled":
		return fmt.Sprintf("Your interview for %s has been scheduled.", jobTitle)
	case "application_selected":
		return fmt.Sprintf("Congratulations. You have been selected for %s.", jobTitle)
	case "application_rejected":
		return fmt.Sprintf("Your application for %s was not selected this time.", jobTitle)
	default:
		return "You have a new update from BlueCollarJob."
	}
}

func stringFromPayload(payload map[string]any, key, fallback string) string {
	value, ok := payload[key].(string)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}
