package service

import (
	"context"
	"encoding/json"
	"errors"
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
	MaxAttempts  int
	RetryBackoff time.Duration
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
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 3
	}
	if cfg.RetryBackoff <= 0 {
		cfg.RetryBackoff = 500 * time.Millisecond
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
		if err := s.sendWithRetry(ctx, event.Recipient, message); err != nil {
			result.Failed++
			if _, markErr := s.notifications.MarkNotificationEventFailed(ctx, event.ID, safeNotificationFailure(err)); markErr != nil {
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

func (s *notificationWorkerService) sendWithRetry(ctx context.Context, recipient, message string) error {
	var lastErr error
	for attempt := 1; attempt <= s.cfg.MaxAttempts; attempt++ {
		if err := s.sender.SendMessage(ctx, recipient, message); err != nil {
			lastErr = err
			if !errors.Is(err, ErrTemporaryWhatsAppDelivery) || attempt == s.cfg.MaxAttempts {
				break
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(s.cfg.RetryBackoff * time.Duration(attempt)):
			}
			continue
		}
		return nil
	}
	return lastErr
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
	case "referral_cashback_pending":
		return "Your Rs 100 referral cashback is pending."
	case "referral_cashback_paid":
		return "Your Rs 100 referral cashback has been paid."
	case "referral_cashback_failed":
		return "Your referral cashback payout failed. Our team will review it."
	case "status_otp":
		return "Your application status OTP is ready. It expires in 5 minutes."
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

func safeNotificationFailure(err error) string {
	if err == nil {
		return ""
	}
	reason := err.Error()
	for _, marker := range []string{"Bearer ", "aadhaar", "otp", "password", "hash"} {
		if strings.Contains(strings.ToLower(reason), strings.ToLower(marker)) {
			return "provider delivery failed"
		}
	}
	if len(reason) > 240 {
		return reason[:240]
	}
	return reason
}
