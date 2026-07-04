package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"bluecollarjob/internal/models"
	"bluecollarjob/internal/repository"
)

const referralCashbackAmountPaise = 10000

type ReferralService interface {
	RegisterReferral(ctx context.Context, referredUser *models.User) (*models.Referral, error)
	CompleteOnboarding(ctx context.Context, referredUserID string) (*models.ReferralTransaction, error)
	GetWorkerReferral(ctx context.Context, workerID string) (*models.User, error)
	ListReferrals(ctx context.Context, workerID string, limit, offset int) ([]models.Referral, error)
	ListTransactions(ctx context.Context, workerID string, limit, offset int) ([]models.ReferralTransaction, error)
	ProcessPendingPayouts(ctx context.Context, limit int) (ReferralPayoutProcessResult, error)
}

type ReferralPayoutGateway interface {
	Payout(ctx context.Context, transaction models.ReferralTransaction) (ReferralPayoutResult, error)
}

type ReferralPayoutResult struct {
	ExternalReference string
}

type ReferralPayoutProcessResult struct {
	Claimed int `json:"claimed"`
	Paid    int `json:"paid"`
	Failed  int `json:"failed"`
}

type referralService struct {
	users         repository.UserRepository
	referrals     repository.ReferralRepository
	notifications repository.NotificationRepository
	payouts       ReferralPayoutGateway
}

func NewReferralService(users repository.UserRepository, referrals repository.ReferralRepository, notifications repository.NotificationRepository, payouts ReferralPayoutGateway) ReferralService {
	return &referralService{
		users:         users,
		referrals:     referrals,
		notifications: notifications,
		payouts:       payouts,
	}
}

func (s *referralService) RegisterReferral(ctx context.Context, referredUser *models.User) (*models.Referral, error) {
	if referredUser == nil || referredUser.ReferredByCode == nil || strings.TrimSpace(*referredUser.ReferredByCode) == "" {
		return nil, nil
	}
	code := strings.TrimSpace(*referredUser.ReferredByCode)
	referrer, err := s.users.GetUserByReferralCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("%w: referred_by_code does not exist", ErrInvalidInput)
	}
	if referrer.ID == referredUser.ID {
		return nil, fmt.Errorf("%w: worker cannot refer self", ErrInvalidInput)
	}
	if existing, err := s.referrals.GetReferralByReferredUserID(ctx, referredUser.ID); err == nil && existing != nil {
		return nil, fmt.Errorf("%w: referral already exists for worker", ErrConflict)
	} else if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}

	return s.referrals.CreateReferral(ctx, &models.Referral{
		ReferrerUserID: referrer.ID,
		ReferredUserID: &referredUser.ID,
		ReferralCode:   code,
	})
}

func (s *referralService) CompleteOnboarding(ctx context.Context, referredUserID string) (*models.ReferralTransaction, error) {
	referral, err := s.referrals.GetReferralByReferredUserID(ctx, referredUserID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	if referral.ConvertedAt != nil {
		return nil, nil
	}
	referral, err = s.referrals.MarkReferralConverted(ctx, referral.ID)
	if err != nil {
		return nil, err
	}
	transaction, err := s.referrals.CreateReferralTransaction(ctx, &models.ReferralTransaction{
		ReferralID:  referral.ID,
		UserID:      referral.ReferrerUserID,
		AmountPaise: referralCashbackAmountPaise,
		Currency:    "INR",
		Status:      "Pending",
	})
	if err != nil {
		return nil, err
	}
	if err := s.createReferralNotification(ctx, referral.ReferrerUserID, "referral_cashback_pending", transaction, "Referral cashback is pending."); err != nil {
		return nil, err
	}
	return transaction, nil
}

func (s *referralService) GetWorkerReferral(ctx context.Context, workerID string) (*models.User, error) {
	return s.users.GetUserByID(ctx, workerID)
}

func (s *referralService) ListReferrals(ctx context.Context, workerID string, limit, offset int) ([]models.Referral, error) {
	return s.referrals.ListReferralsByReferrer(ctx, workerID, limit, offset)
}

func (s *referralService) ListTransactions(ctx context.Context, workerID string, limit, offset int) ([]models.ReferralTransaction, error) {
	return s.referrals.ListReferralTransactionsByUser(ctx, workerID, limit, offset)
}

func (s *referralService) ProcessPendingPayouts(ctx context.Context, limit int) (ReferralPayoutProcessResult, error) {
	transactions, err := s.referrals.ClaimPendingReferralTransactions(ctx, limit)
	if err != nil {
		return ReferralPayoutProcessResult{}, err
	}
	result := ReferralPayoutProcessResult{Claimed: len(transactions)}
	for _, transaction := range transactions {
		payout, err := s.payouts.Payout(ctx, transaction)
		if err != nil {
			result.Failed++
			updated, markErr := s.referrals.MarkReferralTransactionFailed(ctx, transaction.ID, err.Error())
			if markErr != nil {
				return result, markErr
			}
			if err := s.createReferralNotification(ctx, updated.UserID, "referral_cashback_failed", updated, err.Error()); err != nil {
				return result, err
			}
			continue
		}
		result.Paid++
		updated, err := s.referrals.MarkReferralTransactionPaid(ctx, transaction.ID, payout.ExternalReference)
		if err != nil {
			return result, err
		}
		if err := s.createReferralNotification(ctx, updated.UserID, "referral_cashback_paid", updated, "Referral cashback paid."); err != nil {
			return result, err
		}
	}
	return result, nil
}

func (s *referralService) createReferralNotification(ctx context.Context, userID, eventType string, transaction *models.ReferralTransaction, message string) error {
	if s.notifications == nil {
		return nil
	}
	user, err := s.users.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(map[string]any{
		"referral_transaction_id": transaction.ID,
		"amount_paise":            transaction.AmountPaise,
		"currency":                transaction.Currency,
		"status":                  transaction.Status,
		"message":                 message,
	})
	if err != nil {
		return err
	}
	_, err = s.notifications.CreateNotificationEvent(ctx, &models.NotificationEvent{
		UserID:    &user.ID,
		Channel:   "whatsapp",
		EventType: eventType,
		Recipient: user.PhoneNumber,
		Payload:   payload,
		Status:    models.NotificationStatusPending,
	})
	return err
}

type MockReferralPayoutGateway struct {
	Fail bool
}

func NewMockReferralPayoutGateway() *MockReferralPayoutGateway {
	return &MockReferralPayoutGateway{}
}

func (g *MockReferralPayoutGateway) Payout(ctx context.Context, transaction models.ReferralTransaction) (ReferralPayoutResult, error) {
	if g.Fail {
		return ReferralPayoutResult{}, fmt.Errorf("mock payout failed")
	}
	return ReferralPayoutResult{ExternalReference: "mock-payout-" + transaction.ID}, nil
}
