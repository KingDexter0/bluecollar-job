package service

import (
	"context"

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
	applications repository.ApplicationRepository
	jobs         repository.JobRepository
	users        repository.UserRepository
}

func NewApplicationService(applications repository.ApplicationRepository, jobs repository.JobRepository, users repository.UserRepository) ApplicationService {
	return &applicationService{
		applications: applications,
		jobs:         jobs,
		users:        users,
	}
}

func (s *applicationService) CreateApplication(ctx context.Context, input CreateApplicationInput) (*models.Application, error) {
	if _, err := s.users.GetUserByID(ctx, input.UserID); err != nil {
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

	return s.applications.CreateApplication(ctx, &models.Application{
		UserID:     input.UserID,
		JobID:      input.JobID,
		EmployerID: job.EmployerID,
		Status:     models.ApplicationStatusApplied,
		Source:     source,
	})
}

func (s *applicationService) GetApplicationByID(ctx context.Context, id string) (*models.Application, error) {
	return s.applications.GetApplicationByID(ctx, id)
}

func (s *applicationService) ListApplicationsByUser(ctx context.Context, userID string, limit, offset int) ([]models.Application, error) {
	return s.applications.ListApplicationsByUser(ctx, userID, limit, offset)
}
