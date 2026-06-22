package service

import (
	"context"

	"bluecollarjob/internal/models"
	"bluecollarjob/internal/repository"
)

type JobService interface {
	GetJobByID(ctx context.Context, id string) (*models.Job, error)
	ListActiveJobs(ctx context.Context, limit, offset int) ([]models.Job, error)
}

type jobService struct {
	jobs repository.JobRepository
}

func NewJobService(jobs repository.JobRepository) JobService {
	return &jobService{jobs: jobs}
}

func (s *jobService) GetJobByID(ctx context.Context, id string) (*models.Job, error) {
	return s.jobs.GetJobByID(ctx, id)
}

func (s *jobService) ListActiveJobs(ctx context.Context, limit, offset int) ([]models.Job, error) {
	return s.jobs.ListActiveJobs(ctx, limit, offset)
}
