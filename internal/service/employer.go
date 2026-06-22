package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"bluecollarjob/internal/models"
	"bluecollarjob/internal/repository"
)

const growthTierActiveJobLimit = 7

type EmployerService interface {
	Register(ctx context.Context, input RegisterEmployerInput) (*models.Employer, string, error)
	Login(ctx context.Context, email, password string) (*models.Employer, string, error)
	GetProfile(ctx context.Context, employerID string) (*models.Employer, error)
	UpdateProfile(ctx context.Context, employerID string, input UpdateEmployerProfileInput) (*models.Employer, error)
	CreateJob(ctx context.Context, employerID string, input EmployerJobInput) (*models.Job, error)
	ListJobs(ctx context.Context, employerID string, limit, offset int) ([]models.Job, error)
	GetJob(ctx context.Context, employerID, jobID string) (*models.Job, error)
	UpdateJob(ctx context.Context, employerID, jobID string, input EmployerJobInput) (*models.Job, error)
	UpdateJobStatus(ctx context.Context, employerID, jobID string, isActive bool) (*models.Job, error)
}

type RegisterEmployerInput struct {
	CompanyName string
	ContactName string
	Email       string
	Password    string
	PhoneNumber *string
	City        *string
	State       *string
}

type UpdateEmployerProfileInput struct {
	CompanyName string
	ContactName string
	PhoneNumber *string
	City        *string
	State       *string
}

type EmployerJobInput struct {
	Title                    string
	Role                     string
	Description              string
	SkillCategory            string
	LocationCity             string
	LocationState            string
	ShiftSchedule            string
	WageMinPaise             *int
	WageMaxPaise             *int
	RequiredVerificationTier models.VerificationTier
	Openings                 int
	IsActive                 bool
}

type employerService struct {
	employers repository.EmployerRepository
	jobs      repository.JobRepository
	auth      AuthService
}

func NewEmployerService(employers repository.EmployerRepository, jobs repository.JobRepository, auth AuthService) EmployerService {
	return &employerService{employers: employers, jobs: jobs, auth: auth}
}

func (s *employerService) Register(ctx context.Context, input RegisterEmployerInput) (*models.Employer, string, error) {
	passwordHash, err := s.auth.HashPassword(input.Password)
	if err != nil {
		return nil, "", err
	}

	employer, err := s.employers.CreateEmployer(ctx, &models.Employer{
		CompanyName:  strings.TrimSpace(input.CompanyName),
		ContactName:  strings.TrimSpace(input.ContactName),
		Email:        strings.ToLower(strings.TrimSpace(input.Email)),
		PasswordHash: passwordHash,
		PhoneNumber:  cleanStringPtr(input.PhoneNumber),
		City:         cleanStringPtr(input.City),
		State:        cleanStringPtr(input.State),
		IsVerified:   false,
	})
	if err != nil {
		return nil, "", err
	}

	token, err := s.auth.GenerateEmployerToken(employer.ID)
	if err != nil {
		return nil, "", err
	}
	return employer, token, nil
}

func (s *employerService) Login(ctx context.Context, email, password string) (*models.Employer, string, error) {
	employer, err := s.employers.GetEmployerByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if err != nil {
		return nil, "", err
	}
	if err := s.auth.CheckPassword(password, employer.PasswordHash); err != nil {
		return nil, "", err
	}
	token, err := s.auth.GenerateEmployerToken(employer.ID)
	if err != nil {
		return nil, "", err
	}
	return employer, token, nil
}

func (s *employerService) GetProfile(ctx context.Context, employerID string) (*models.Employer, error) {
	return s.employers.GetEmployerByID(ctx, employerID)
}

func (s *employerService) UpdateProfile(ctx context.Context, employerID string, input UpdateEmployerProfileInput) (*models.Employer, error) {
	employer, err := s.employers.GetEmployerByID(ctx, employerID)
	if err != nil {
		return nil, err
	}
	employer.CompanyName = strings.TrimSpace(input.CompanyName)
	employer.ContactName = strings.TrimSpace(input.ContactName)
	employer.PhoneNumber = cleanStringPtr(input.PhoneNumber)
	employer.City = cleanStringPtr(input.City)
	employer.State = cleanStringPtr(input.State)
	return s.employers.UpdateEmployerProfile(ctx, employer)
}

func (s *employerService) CreateJob(ctx context.Context, employerID string, input EmployerJobInput) (*models.Job, error) {
	if input.IsActive {
		if err := s.ensureCanActivateJob(ctx, employerID, ""); err != nil {
			return nil, err
		}
	}

	return s.jobs.CreateJob(ctx, buildEmployerJob(employerID, "", input))
}

func (s *employerService) ListJobs(ctx context.Context, employerID string, limit, offset int) ([]models.Job, error) {
	return s.jobs.ListJobsByEmployer(ctx, employerID, limit, offset)
}

func (s *employerService) GetJob(ctx context.Context, employerID, jobID string) (*models.Job, error) {
	return s.jobs.GetJobByIDAndEmployer(ctx, jobID, employerID)
}

func (s *employerService) UpdateJob(ctx context.Context, employerID, jobID string, input EmployerJobInput) (*models.Job, error) {
	current, err := s.jobs.GetJobByIDAndEmployer(ctx, jobID, employerID)
	if err != nil {
		return nil, err
	}
	if input.IsActive && !current.IsActive {
		if err := s.ensureCanActivateJob(ctx, employerID, jobID); err != nil {
			return nil, err
		}
	}
	job := buildEmployerJob(employerID, jobID, input)
	return s.jobs.UpdateJob(ctx, job)
}

func (s *employerService) UpdateJobStatus(ctx context.Context, employerID, jobID string, isActive bool) (*models.Job, error) {
	if isActive {
		current, err := s.jobs.GetJobByIDAndEmployer(ctx, jobID, employerID)
		if err != nil {
			return nil, err
		}
		if !current.IsActive {
			if err := s.ensureCanActivateJob(ctx, employerID, jobID); err != nil {
				return nil, err
			}
		}
	}
	return s.jobs.UpdateEmployerJobStatus(ctx, jobID, employerID, isActive)
}

func (s *employerService) ensureCanActivateJob(ctx context.Context, employerID, jobID string) error {
	tier, err := s.employers.GetActiveSubscriptionTier(ctx, employerID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			tier = models.SubscriptionTierGrowth
		} else {
			return err
		}
	}
	if tier == models.SubscriptionTierEnterprise {
		return nil
	}
	count, err := s.jobs.CountActiveJobsByEmployer(ctx, employerID)
	if err != nil {
		return err
	}
	if count >= growthTierActiveJobLimit {
		return fmt.Errorf("%w: Growth tier employers can have maximum %d active jobs", ErrConflict, growthTierActiveJobLimit)
	}
	return nil
}

func buildEmployerJob(employerID, jobID string, input EmployerJobInput) *models.Job {
	tier := input.RequiredVerificationTier
	if tier == "" {
		tier = models.VerificationTierLow
	}
	openings := input.Openings
	if openings <= 0 {
		openings = 1
	}
	skillCategory := strings.TrimSpace(input.SkillCategory)
	if skillCategory == "" {
		skillCategory = strings.TrimSpace(input.Role)
	}
	return &models.Job{
		ID:                       jobID,
		EmployerID:               employerID,
		Title:                    strings.TrimSpace(input.Title),
		Role:                     strings.TrimSpace(input.Role),
		Description:              strings.TrimSpace(input.Description),
		SkillCategory:            skillCategory,
		LocationCity:             strings.TrimSpace(input.LocationCity),
		LocationState:            strings.TrimSpace(input.LocationState),
		ShiftSchedule:            strings.TrimSpace(input.ShiftSchedule),
		WageMinPaise:             input.WageMinPaise,
		WageMaxPaise:             input.WageMaxPaise,
		RequiredVerificationTier: tier,
		Openings:                 openings,
		IsActive:                 input.IsActive,
	}
}
