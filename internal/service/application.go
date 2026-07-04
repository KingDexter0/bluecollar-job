package service

import (
	"context"
	"encoding/json"

	"bluecollarjob/internal/models"
	"bluecollarjob/internal/repository"
)

type ApplicationService interface {
	CreateApplication(ctx context.Context, input CreateApplicationInput) (*models.Application, error)
	GetApplicationByID(ctx context.Context, id string) (*models.Application, error)
	ListApplicationsByUser(ctx context.Context, userID string, limit, offset int) ([]models.Application, error)
}

type CreateApplicationInput struct {
	UserID string
	JobID  string
	Source string
}

type applicationService struct {
	applications  repository.ApplicationRepository
	jobs          repository.JobRepository
	users         repository.UserRepository
	notifications repository.NotificationRepository
}

func NewApplicationService(applications repository.ApplicationRepository, jobs repository.JobRepository, users repository.UserRepository, notifications repository.NotificationRepository) ApplicationService {
	return &applicationService{
		applications:  applications,
		jobs:          jobs,
		users:         users,
		notifications: notifications,
	}
}

func (s *applicationService) CreateApplication(ctx context.Context, input CreateApplicationInput) (*models.Application, error) {
	user, err := s.users.GetUserByID(ctx, input.UserID)
	if err != nil {
		return nil, err
	}

	job, err := s.jobs.GetJobByID(ctx, input.JobID)
	if err != nil {
		return nil, err
	}

	source := input.Source
	if source == "" {
		source = "api"
	}

	application, err := s.applications.CreateApplication(ctx, &models.Application{
		UserID:     input.UserID,
		JobID:      input.JobID,
		EmployerID: job.EmployerID,
		Status:     models.ApplicationStatusApplied,
		Source:     source,
	})
	if err != nil {
		return nil, err
	}

	if s.notifications != nil {
		payload, err := json.Marshal(map[string]any{
			"application_id": application.ID,
			"job_id":         application.JobID,
			"job_title":      job.Title,
			"job_role":       job.Role,
			"status":         application.Status,
		})
		if err != nil {
			return nil, err
		}
		if _, err := s.notifications.CreateNotificationEvent(ctx, &models.NotificationEvent{
			UserID:        &application.UserID,
			EmployerID:    &application.EmployerID,
			ApplicationID: &application.ID,
			Channel:       "whatsapp",
			EventType:     "application_submitted",
			Recipient:     user.PhoneNumber,
			Payload:       payload,
			Status:        models.NotificationStatusPending,
		}); err != nil {
			return nil, err
		}
	}

	return application, nil
}

func (s *applicationService) GetApplicationByID(ctx context.Context, id string) (*models.Application, error) {
	return s.applications.GetApplicationByID(ctx, id)
}

func (s *applicationService) ListApplicationsByUser(ctx context.Context, userID string, limit, offset int) ([]models.Application, error) {
	return s.applications.ListApplicationsByUser(ctx, userID, limit, offset)
}
