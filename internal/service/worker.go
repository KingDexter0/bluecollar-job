package service

import (
	"context"
	"strings"

	"bluecollarjob/internal/models"
	"bluecollarjob/internal/repository"
)

type WorkerService interface {
	CreateWorker(ctx context.Context, input CreateWorkerInput) (*models.User, error)
	GetWorkerByID(ctx context.Context, id string) (*models.User, error)
	GetWorkerByPhone(ctx context.Context, phoneNumber string) (*models.User, error)
	UpdateWorkerProfile(ctx context.Context, id string, input UpdateWorkerProfileInput) (*models.User, error)
	GetWorkerByReferralCode(ctx context.Context, referralCode string) (*models.User, error)
}

type CreateWorkerInput struct {
	PhoneNumber        string
	FullName           string
	LanguagePreference string
	TargetRole         *string
	PreferredZone      *string
	ReferralCode       string
	ReferredByCode     *string
}

type UpdateWorkerProfileInput struct {
	FullName           string
	LanguagePreference string
	TargetRole         *string
	PreferredZone      *string
}

type workerService struct {
	users repository.UserRepository
}

func NewWorkerService(users repository.UserRepository) WorkerService {
	return &workerService{users: users}
}

func (s *workerService) CreateWorker(ctx context.Context, input CreateWorkerInput) (*models.User, error) {
	language := strings.TrimSpace(input.LanguagePreference)
	if language == "" {
		language = "en"
	}

	user, err := s.users.CreateUser(ctx, &models.User{
		PhoneNumber:        strings.TrimSpace(input.PhoneNumber),
		FullName:           strings.TrimSpace(input.FullName),
		LanguagePreference: language,
		TargetRole:         cleanStringPtr(input.TargetRole),
		PreferredZone:      cleanStringPtr(input.PreferredZone),
		VerificationTier:   models.VerificationTierHigh,
		ReferralCode:       strings.TrimSpace(input.ReferralCode),
		ReferredByCode:     cleanStringPtr(input.ReferredByCode),
		IsActive:           true,
	})
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *workerService) GetWorkerByID(ctx context.Context, id string) (*models.User, error) {
	return s.users.GetUserByID(ctx, id)
}

func (s *workerService) GetWorkerByPhone(ctx context.Context, phoneNumber string) (*models.User, error) {
	return s.users.GetUserByPhone(ctx, phoneNumber)
}

func (s *workerService) UpdateWorkerProfile(ctx context.Context, id string, input UpdateWorkerProfileInput) (*models.User, error) {
	current, err := s.users.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}

	language := strings.TrimSpace(input.LanguagePreference)
	if language == "" {
		language = current.LanguagePreference
	}

	current.FullName = strings.TrimSpace(input.FullName)
	current.LanguagePreference = language
	current.TargetRole = cleanStringPtr(input.TargetRole)
	current.PreferredZone = cleanStringPtr(input.PreferredZone)

	return s.users.UpdateUserProfile(ctx, current)
}

func (s *workerService) GetWorkerByReferralCode(ctx context.Context, referralCode string) (*models.User, error) {
	return s.users.GetUserByReferralCode(ctx, referralCode)
}

func cleanStringPtr(value *string) *string {
	if value == nil {
		return nil
	}
	cleaned := strings.TrimSpace(*value)
	if cleaned == "" {
		return nil
	}
	return &cleaned
}
