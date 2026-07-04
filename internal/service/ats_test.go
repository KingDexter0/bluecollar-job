package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"bluecollarjob/internal/models"
	"bluecollarjob/internal/repository"
)

func TestATSServiceStatusUpdateCreatesNotification(t *testing.T) {
	ctx := context.Background()
	notifications := &fakeNotificationRepository{}
	service := NewATSService(&fakeATSRepository{}, notifications)

	application, err := service.UpdateApplicationStatus(ctx, "employer-1", "application-1", models.ApplicationStatusShortlisted)
	if err != nil {
		t.Fatalf("update status: %v", err)
	}
	if application.Status != models.ApplicationStatusShortlisted {
		t.Fatalf("expected shortlisted, got %s", application.Status)
	}
	if len(notifications.events) != 1 {
		t.Fatalf("expected one notification event, got %d", len(notifications.events))
	}
	if notifications.events[0].EventType != "application_shortlisted" {
		t.Fatalf("expected shortlisted notification, got %s", notifications.events[0].EventType)
	}
}

func TestATSServiceDirectInterviewCreatesNotification(t *testing.T) {
	ctx := context.Background()
	notifications := &fakeNotificationRepository{}
	service := NewATSService(&fakeATSRepository{}, notifications)

	slot, application, err := service.ScheduleDirectInterview(ctx, "employer-1", "application-1", validInterviewInput())
	if err != nil {
		t.Fatalf("schedule interview: %v", err)
	}
	if slot.Status != models.InterviewSlotStatusConfirmed {
		t.Fatalf("expected confirmed slot, got %s", slot.Status)
	}
	if application.Status != models.ApplicationStatusInterviewScheduled {
		t.Fatalf("expected interview scheduled, got %s", application.Status)
	}
	if len(notifications.events) != 1 || notifications.events[0].EventType != "interview_scheduled" {
		t.Fatalf("expected interview notification, got %#v", notifications.events)
	}
}

func TestATSServiceCreateInterviewSlotsRequiresThreeSlots(t *testing.T) {
	ctx := context.Background()
	service := NewATSService(&fakeATSRepository{}, &fakeNotificationRepository{})

	_, _, err := service.CreateInterviewSlots(ctx, "employer-1", "application-1", []InterviewScheduleInput{validInterviewInput()})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid input, got %v", err)
	}
}

func TestATSServiceDuplicateSlotSelectionMapsToConflict(t *testing.T) {
	ctx := context.Background()
	service := NewATSService(&fakeATSRepository{selectErr: repository.ErrConflict}, &fakeNotificationRepository{})

	_, _, err := service.SelectInterviewSlot(ctx, "application-1", "slot-1")
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected service conflict, got %v", err)
	}
}

func validInterviewInput() InterviewScheduleInput {
	factoryLocation := "ACME Factory, Gurugram"
	googleMapsURL := "https://maps.google.com/?q=ACME+Factory+Gurugram"
	startsAt := time.Now().UTC().Add(24 * time.Hour).Truncate(time.Second)
	return InterviewScheduleInput{
		StartsAt:        startsAt,
		EndsAt:          startsAt.Add(time.Hour),
		Timezone:        "Asia/Kolkata",
		FactoryLocation: &factoryLocation,
		GoogleMapsURL:   &googleMapsURL,
	}
}

type fakeATSRepository struct {
	selectErr error
}

func (r *fakeATSRepository) ListEmployerApplications(ctx context.Context, employerID string, filters repository.EmployerApplicationFilters) ([]models.ApplicationATS, error) {
	return []models.ApplicationATS{fakeATSApplication(models.ApplicationStatusApplied)}, nil
}

func (r *fakeATSRepository) GetEmployerApplicationByID(ctx context.Context, employerID, applicationID string) (*models.ApplicationATS, error) {
	application := fakeATSApplication(models.ApplicationStatusInterviewScheduled)
	return &application, nil
}

func (r *fakeATSRepository) UpdateEmployerApplicationStatus(ctx context.Context, employerID, applicationID string, status models.ApplicationStatus) (*models.ApplicationATS, error) {
	application := fakeATSApplication(status)
	return &application, nil
}

func (r *fakeATSRepository) CreateDirectInterview(ctx context.Context, employerID, applicationID string, slot repository.InterviewSlotInput) (*models.InterviewSlot, *models.ApplicationATS, error) {
	application := fakeATSApplication(models.ApplicationStatusInterviewScheduled)
	return &models.InterviewSlot{ID: "slot-1", ApplicationID: applicationID, Status: models.InterviewSlotStatusConfirmed, StartsAt: slot.StartsAt, EndsAt: slot.EndsAt}, &application, nil
}

func (r *fakeATSRepository) CreateInterviewSlots(ctx context.Context, employerID, applicationID string, slots []repository.InterviewSlotInput) ([]models.InterviewSlot, *models.ApplicationATS, error) {
	application := fakeATSApplication(models.ApplicationStatusSlotSelectionPending)
	created := make([]models.InterviewSlot, 0, len(slots))
	for i, slot := range slots {
		created = append(created, models.InterviewSlot{
			ID:            string(rune('a' + i)),
			ApplicationID: applicationID,
			Status:        models.InterviewSlotStatusAvailable,
			StartsAt:      slot.StartsAt,
			EndsAt:        slot.EndsAt,
		})
	}
	return created, &application, nil
}

func (r *fakeATSRepository) SelectInterviewSlot(ctx context.Context, applicationID, slotID string) (*models.InterviewSlot, *models.Application, error) {
	if r.selectErr != nil {
		return nil, nil, r.selectErr
	}
	return &models.InterviewSlot{ID: slotID, ApplicationID: applicationID, Status: models.InterviewSlotStatusConfirmed}, &models.Application{ID: applicationID, EmployerID: "employer-1"}, nil
}

type fakeNotificationRepository struct {
	events []models.NotificationEvent
}

func (r *fakeNotificationRepository) CreateNotificationEvent(ctx context.Context, event *models.NotificationEvent) (*models.NotificationEvent, error) {
	event.ID = "notification-1"
	r.events = append(r.events, *event)
	return event, nil
}

func (r *fakeNotificationRepository) ClaimPendingNotificationEvents(ctx context.Context, limit int) ([]models.NotificationEvent, error) {
	return nil, nil
}

func (r *fakeNotificationRepository) ListNotificationEvents(ctx context.Context, filters repository.NotificationEventFilters) ([]models.NotificationEvent, error) {
	return nil, nil
}

func (r *fakeNotificationRepository) MarkNotificationEventSent(ctx context.Context, id string) (*models.NotificationEvent, error) {
	return &models.NotificationEvent{ID: id, Status: models.NotificationStatusSent}, nil
}

func (r *fakeNotificationRepository) MarkNotificationEventFailed(ctx context.Context, id string, reason string) (*models.NotificationEvent, error) {
	return &models.NotificationEvent{ID: id, Status: models.NotificationStatusFailed, LastError: &reason}, nil
}

func fakeATSApplication(status models.ApplicationStatus) models.ApplicationATS {
	return models.ApplicationATS{
		Application: models.Application{
			ID:         "application-1",
			UserID:     "worker-1",
			JobID:      "job-1",
			EmployerID: "employer-1",
			Status:     status,
			Source:     "test",
		},
		WorkerFullName:         "Test Worker",
		WorkerPhoneNumber:      "+919876543210",
		WorkerVerificationTier: models.VerificationTierLow,
		JobTitle:               "Machine Operator",
		JobRole:                "Machine Operator",
	}
}
