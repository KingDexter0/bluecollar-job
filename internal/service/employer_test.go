package service

import (
	"context"
	"errors"
	"testing"

	"bluecollarjob/internal/models"
	"bluecollarjob/internal/repository"
)

func TestEmployerServiceGrowthTierActiveJobLimit(t *testing.T) {
	ctx := context.Background()
	employers := &fakeEmployerRepository{tier: models.SubscriptionTierGrowth}
	jobs := &fakeJobRepository{activeCount: 7}
	service := NewEmployerService(employers, jobs, NewAuthService("test-secret", "test"))

	_, err := service.CreateJob(ctx, "employer-1", validEmployerJobInput(true))
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected ErrConflict for growth limit, got %v", err)
	}
}

func TestEmployerServiceEnterpriseAllowsUnlimitedJobs(t *testing.T) {
	ctx := context.Background()
	employers := &fakeEmployerRepository{tier: models.SubscriptionTierEnterprise}
	jobs := &fakeJobRepository{activeCount: 100}
	service := NewEmployerService(employers, jobs, NewAuthService("test-secret", "test"))

	job, err := service.CreateJob(ctx, "employer-1", validEmployerJobInput(true))
	if err != nil {
		t.Fatalf("expected enterprise job creation to pass: %v", err)
	}
	if job.ID == "" {
		t.Fatal("expected created job id")
	}
}

func validEmployerJobInput(active bool) EmployerJobInput {
	return EmployerJobInput{
		Title:         "Machine Operator",
		Role:          "Machine Operator",
		Description:   "Operate factory equipment.",
		SkillCategory: "Manufacturing",
		LocationCity:  "Pune",
		LocationState: "Maharashtra",
		ShiftSchedule: "Day shift",
		Openings:      1,
		IsActive:      active,
	}
}

type fakeEmployerRepository struct {
	tier models.SubscriptionTier
}

func (r *fakeEmployerRepository) CreateEmployer(ctx context.Context, employer *models.Employer) (*models.Employer, error) {
	employer.ID = "employer-1"
	return employer, nil
}

func (r *fakeEmployerRepository) GetEmployerByID(ctx context.Context, id string) (*models.Employer, error) {
	return &models.Employer{ID: id, CompanyName: "Test", ContactName: "Test", Email: "test@example.com"}, nil
}

func (r *fakeEmployerRepository) GetEmployerByEmail(ctx context.Context, email string) (*models.Employer, error) {
	hash, _ := NewAuthService("test-secret", "test").HashPassword("password")
	return &models.Employer{ID: "employer-1", Email: email, PasswordHash: hash}, nil
}

func (r *fakeEmployerRepository) UpdateEmployerProfile(ctx context.Context, employer *models.Employer) (*models.Employer, error) {
	return employer, nil
}

func (r *fakeEmployerRepository) GetActiveSubscriptionTier(ctx context.Context, employerID string) (models.SubscriptionTier, error) {
	if r.tier == "" {
		return "", repository.ErrNotFound
	}
	return r.tier, nil
}

type fakeJobRepository struct {
	activeCount int
}

func (r *fakeJobRepository) CreateJob(ctx context.Context, job *models.Job) (*models.Job, error) {
	job.ID = "job-1"
	return job, nil
}

func (r *fakeJobRepository) GetJobByID(ctx context.Context, id string) (*models.Job, error) {
	return &models.Job{ID: id, EmployerID: "employer-1", IsActive: true}, nil
}

func (r *fakeJobRepository) GetJobByIDAndEmployer(ctx context.Context, id, employerID string) (*models.Job, error) {
	return &models.Job{ID: id, EmployerID: employerID, IsActive: false}, nil
}

func (r *fakeJobRepository) ListActiveJobs(ctx context.Context, limit, offset int) ([]models.Job, error) {
	return nil, nil
}

func (r *fakeJobRepository) ListJobsByEmployer(ctx context.Context, employerID string, limit, offset int) ([]models.Job, error) {
	return nil, nil
}

func (r *fakeJobRepository) CountActiveJobsByEmployer(ctx context.Context, employerID string) (int, error) {
	return r.activeCount, nil
}

func (r *fakeJobRepository) UpdateJob(ctx context.Context, job *models.Job) (*models.Job, error) {
	return job, nil
}

func (r *fakeJobRepository) UpdateJobStatus(ctx context.Context, id string, isActive bool) (*models.Job, error) {
	return &models.Job{ID: id, IsActive: isActive}, nil
}

func (r *fakeJobRepository) UpdateEmployerJobStatus(ctx context.Context, id, employerID string, isActive bool) (*models.Job, error) {
	return &models.Job{ID: id, EmployerID: employerID, IsActive: isActive}, nil
}
