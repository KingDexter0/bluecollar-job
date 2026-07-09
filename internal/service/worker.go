package service

import (
	"context"
	"fmt"
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
	users     repository.UserRepository
	referrals ReferralService
}

func NewWorkerService(users repository.UserRepository, referrals ...ReferralService) WorkerService {
	var referralService ReferralService
	if len(referrals) > 0 {
		referralService = referrals[0]
	}
	return &workerService{users: users, referrals: referralService}
}

func (s *workerService) CreateWorker(ctx context.Context, input CreateWorkerInput) (*models.User, error) {
	language := strings.TrimSpace(input.LanguagePreference)
	if language == "" {
		language = "en"
	}
	referralCode := strings.TrimSpace(input.ReferralCode)
	if referralCode == "" {
		referralCode = referralCodeFromPhone(input.PhoneNumber)
	}
	referredByCode := cleanStringPtr(input.ReferredByCode)
	if s.referrals != nil && referredByCode != nil {
		code := strings.TrimSpace(*referredByCode)
		if code == referralCode {
			return nil, fmt.Errorf("%w: worker cannot refer self", ErrInvalidInput)
		}
		if _, err := s.users.GetUserByReferralCode(ctx, code); err != nil {
			return nil, fmt.Errorf("%w: referred_by_code does not exist", ErrInvalidInput)
		}
	}

	user, err := s.users.CreateUser(ctx, &models.User{
		PhoneNumber:        strings.TrimSpace(input.PhoneNumber),
		FullName:           strings.TrimSpace(input.FullName),
		LanguagePreference: language,
		TargetRole:         cleanStringPtr(input.TargetRole),
		PreferredZone:      cleanStringPtr(input.PreferredZone),
		VerificationTier:   models.VerificationTierHigh,
		ReferralCode:       referralCode,
		ReferredByCode:     referredByCode,
		IsActive:           true,
	})
	if err != nil {
		return nil, err
	}
	if s.referrals != nil {
		if _, err := s.referrals.RegisterReferral(ctx, user); err != nil {
			return nil, err
		}
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
