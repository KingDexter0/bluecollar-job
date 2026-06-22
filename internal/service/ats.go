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

type ATSService interface {
	ListEmployerApplications(ctx context.Context, employerID string, filters repository.EmployerApplicationFilters) ([]models.ApplicationATS, error)
	GetEmployerApplication(ctx context.Context, employerID, applicationID string) (*models.ApplicationATS, error)
	UpdateApplicationStatus(ctx context.Context, employerID, applicationID string, status models.ApplicationStatus) (*models.ApplicationATS, error)
	ScheduleDirectInterview(ctx context.Context, employerID, applicationID string, input InterviewScheduleInput) (*models.InterviewSlot, *models.ApplicationATS, error)
	CreateInterviewSlots(ctx context.Context, employerID, applicationID string, slots []InterviewScheduleInput) ([]models.InterviewSlot, *models.ApplicationATS, error)
	SelectInterviewSlot(ctx context.Context, applicationID, slotID string) (*models.InterviewSlot, *models.Application, error)
}

type InterviewScheduleInput struct {
	StartsAt        time.Time
	EndsAt          time.Time
	Timezone        string
	FactoryLocation *string
	GoogleMapsURL   *string
}

type atsService struct {
	ats           repository.ATSRepository
	notifications repository.NotificationRepository
}

func NewATSService(ats repository.ATSRepository, notifications repository.NotificationRepository) ATSService {
	return &atsService{ats: ats, notifications: notifications}
}

func (s *atsService) ListEmployerApplications(ctx context.Context, employerID string, filters repository.EmployerApplicationFilters) ([]models.ApplicationATS, error) {
	return s.ats.ListEmployerApplications(ctx, employerID, filters)
}

func (s *atsService) GetEmployerApplication(ctx context.Context, employerID, applicationID string) (*models.ApplicationATS, error) {
	return s.ats.GetEmployerApplicationByID(ctx, employerID, applicationID)
}

func (s *atsService) UpdateApplicationStatus(ctx context.Context, employerID, applicationID string, status models.ApplicationStatus) (*models.ApplicationATS, error) {
	if !validApplicationStatus(status) {
		return nil, fmt.Errorf("%w: invalid application status", ErrInvalidInput)
	}

	application, err := s.ats.UpdateEmployerApplicationStatus(ctx, employerID, applicationID, status)
	if err != nil {
		return nil, err
	}
	return application, s.createStatusNotification(ctx, application, status)
}

func (s *atsService) ScheduleDirectInterview(ctx context.Context, employerID, applicationID string, input InterviewScheduleInput) (*models.InterviewSlot, *models.ApplicationATS, error) {
	if err := validateInterviewSchedule(input); err != nil {
		return nil, nil, err
	}

	slot, application, err := s.ats.CreateDirectInterview(ctx, employerID, applicationID, toRepositorySlot(input))
	if err != nil {
		return nil, nil, err
	}
	return slot, application, s.createStatusNotification(ctx, application, models.ApplicationStatusInterviewScheduled)
}

func (s *atsService) CreateInterviewSlots(ctx context.Context, employerID, applicationID string, slots []InterviewScheduleInput) ([]models.InterviewSlot, *models.ApplicationATS, error) {
	if len(slots) != 3 {
		return nil, nil, fmt.Errorf("%w: exactly 3 interview slots are required", ErrInvalidInput)
	}

	repositorySlots := make([]repository.InterviewSlotInput, 0, len(slots))
	for _, slot := range slots {
		if err := validateInterviewSchedule(slot); err != nil {
			return nil, nil, err
		}
		repositorySlots = append(repositorySlots, toRepositorySlot(slot))
	}

	createdSlots, application, err := s.ats.CreateInterviewSlots(ctx, employerID, applicationID, repositorySlots)
	if err != nil {
		return nil, nil, err
	}
	return createdSlots, application, s.createStatusNotification(ctx, application, models.ApplicationStatusSlotSelectionPending)
}

func (s *atsService) SelectInterviewSlot(ctx context.Context, applicationID, slotID string) (*models.InterviewSlot, *models.Application, error) {
	slot, application, err := s.ats.SelectInterviewSlot(ctx, applicationID, slotID)
	if err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return nil, nil, fmt.Errorf("%w: interview slot is no longer available", ErrConflict)
		}
		return nil, nil, err
	}

	atsApplication, err := s.ats.GetEmployerApplicationByID(ctx, application.EmployerID, application.ID)
	if err != nil {
		return nil, nil, err
	}
	if err := s.createStatusNotification(ctx, atsApplication, models.ApplicationStatusInterviewScheduled); err != nil {
		return nil, nil, err
	}
	return slot, application, nil
}

func (s *atsService) createStatusNotification(ctx context.Context, application *models.ApplicationATS, status models.ApplicationStatus) error {
	eventType, ok := notificationEventType(status)
	if !ok {
		return nil
	}

	payload, err := json.Marshal(map[string]any{
		"application_id": application.ID,
		"job_id":         application.JobID,
		"job_title":      application.JobTitle,
		"job_role":       application.JobRole,
		"status":         status,
	})
	if err != nil {
		return err
	}

	_, err = s.notifications.CreateNotificationEvent(ctx, &models.NotificationEvent{
		UserID:        &application.UserID,
		EmployerID:    &application.EmployerID,
		ApplicationID: &application.ID,
		Channel:       "whatsapp",
		EventType:     eventType,
		Recipient:     application.WorkerPhoneNumber,
		Payload:       payload,
		Status:        models.NotificationStatusPending,
	})
	return err
}

func validApplicationStatus(status models.ApplicationStatus) bool {
	switch status {
	case models.ApplicationStatusApplied,
		models.ApplicationStatusShortlisted,
		models.ApplicationStatusSlotSelectionPending,
		models.ApplicationStatusInterviewScheduled,
		models.ApplicationStatusSelected,
		models.ApplicationStatusRejected:
		return true
	default:
		return false
	}
}

func validVerificationTier(tier models.VerificationTier) bool {
	switch tier {
	case models.VerificationTierLow, models.VerificationTierMedium, models.VerificationTierHigh:
		return true
	default:
		return false
	}
}

func notificationEventType(status models.ApplicationStatus) (string, bool) {
	switch status {
	case models.ApplicationStatusShortlisted:
		return "application_shortlisted", true
	case models.ApplicationStatusSlotSelectionPending:
		return "interview_slot_selection_pending", true
	case models.ApplicationStatusInterviewScheduled:
		return "interview_scheduled", true
	case models.ApplicationStatusSelected:
		return "application_selected", true
	case models.ApplicationStatusRejected:
		return "application_rejected", true
	default:
		return "", false
	}
}

func validateInterviewSchedule(input InterviewScheduleInput) error {
	if input.StartsAt.IsZero() || input.EndsAt.IsZero() {
		return fmt.Errorf("%w: starts_at and ends_at are required", ErrInvalidInput)
	}
	if !input.StartsAt.Before(input.EndsAt) {
		return fmt.Errorf("%w: starts_at must be before ends_at", ErrInvalidInput)
	}
	if input.FactoryLocation == nil || strings.TrimSpace(*input.FactoryLocation) == "" {
		return fmt.Errorf("%w: factory_location is required", ErrInvalidInput)
	}
	if input.GoogleMapsURL == nil || strings.TrimSpace(*input.GoogleMapsURL) == "" {
		return fmt.Errorf("%w: google_maps_url is required", ErrInvalidInput)
	}
	return nil
}

func toRepositorySlot(input InterviewScheduleInput) repository.InterviewSlotInput {
	timezone := strings.TrimSpace(input.Timezone)
	if timezone == "" {
		timezone = "Asia/Kolkata"
	}
	return repository.InterviewSlotInput{
		StartsAt:        input.StartsAt,
		EndsAt:          input.EndsAt,
		Timezone:        timezone,
		FactoryLocation: cleanStringPtr(input.FactoryLocation),
		GoogleMapsURL:   cleanStringPtr(input.GoogleMapsURL),
	}
}
