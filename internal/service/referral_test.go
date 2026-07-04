package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"bluecollarjob/internal/models"
	"bluecollarjob/internal/repository"
)

func TestWorkerServiceGeneratesReferralCode(t *testing.T) {
	ctx := context.Background()
	users := newReferralUserRepository()
	workers := NewWorkerService(users)

	worker, err := workers.CreateWorker(ctx, CreateWorkerInput{
		PhoneNumber: "+919876540001",
		FullName:    "Referral Code Worker",
	})
	if err != nil {
		t.Fatalf("create worker: %v", err)
	}
	if worker.ReferralCode == "" {
		t.Fatal("expected referral code")
	}
	if _, err := users.GetUserByReferralCode(ctx, worker.ReferralCode); err != nil {
		t.Fatalf("expected referral code lookup: %v", err)
	}
}

func TestReferralServiceValidReferralCreation(t *testing.T) {
	ctx := context.Background()
	users := newReferralUserRepository()
	referrer := users.mustCreate("+919876540010", "Referrer", "REFER100", nil)
	code := referrer.ReferralCode
	referred := users.mustCreate("+919876540011", "Referred", "REFERRED100", &code)
	referrals := newReferralRepository()
	service := NewReferralService(users, referrals, newReferralNotificationRepository(), NewMockReferralPayoutGateway())

	referral, err := service.RegisterReferral(ctx, referred)
	if err != nil {
		t.Fatalf("register referral: %v", err)
	}
	if referral.ReferrerUserID != referrer.ID || referral.ReferredUserID == nil || *referral.ReferredUserID != referred.ID {
		t.Fatalf("unexpected referral: %#v", referral)
	}
}

func TestReferralServiceInvalidReferralCodeRejected(t *testing.T) {
	ctx := context.Background()
	users := newReferralUserRepository()
	badCode := "MISSING"
	referred := users.mustCreate("+919876540012", "Referred", "REFERRED101", &badCode)
	service := NewReferralService(users, newReferralRepository(), newReferralNotificationRepository(), NewMockReferralPayoutGateway())

	_, err := service.RegisterReferral(ctx, referred)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid input, got %v", err)
	}
}

func TestReferralServiceSelfReferralPrevention(t *testing.T) {
	ctx := context.Background()
	users := newReferralUserRepository()
	code := "SELF100"
	user := users.mustCreate("+919876540013", "Self", code, &code)
	service := NewReferralService(users, newReferralRepository(), newReferralNotificationRepository(), NewMockReferralPayoutGateway())

	_, err := service.RegisterReferral(ctx, user)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected self referral invalid input, got %v", err)
	}
}

func TestReferralServiceCashbackTransactionAfterOnboarding(t *testing.T) {
	ctx := context.Background()
	users := newReferralUserRepository()
	referrer := users.mustCreate("+919876540014", "Referrer", "REFER101", nil)
	code := referrer.ReferralCode
	referred := users.mustCreate("+919876540015", "Referred", "REFERRED102", &code)
	referrals := newReferralRepository()
	notifications := newReferralNotificationRepository()
	service := NewReferralService(users, referrals, notifications, NewMockReferralPayoutGateway())
	if _, err := service.RegisterReferral(ctx, referred); err != nil {
		t.Fatalf("register referral: %v", err)
	}

	transaction, err := service.CompleteOnboarding(ctx, referred.ID)
	if err != nil {
		t.Fatalf("complete onboarding: %v", err)
	}
	if transaction.AmountPaise != referralCashbackAmountPaise || transaction.Currency != "INR" || transaction.Status != "Pending" {
		t.Fatalf("unexpected transaction: %#v", transaction)
	}
	if len(notifications.events) != 1 || notifications.events[0].EventType != "referral_cashback_pending" {
		t.Fatalf("expected pending notification, got %#v", notifications.events)
	}
}

func TestReferralServiceMockPayoutSuccess(t *testing.T) {
	ctx := context.Background()
	users := newReferralUserRepository()
	referrer := users.mustCreate("+919876540016", "Referrer", "REFER102", nil)
	referrals := newReferralRepository()
	notifications := newReferralNotificationRepository()
	transaction := referrals.mustTransaction(referrer.ID, "Pending")
	service := NewReferralService(users, referrals, notifications, NewMockReferralPayoutGateway())

	result, err := service.ProcessPendingPayouts(ctx, 10)
	if err != nil {
		t.Fatalf("process payouts: %v", err)
	}
	if result.Paid != 1 || referrals.transactions[transaction.ID].Status != "Paid" {
		t.Fatalf("expected paid result, got result=%#v transaction=%#v", result, referrals.transactions[transaction.ID])
	}
}

func TestReferralServiceMockPayoutFailure(t *testing.T) {
	ctx := context.Background()
	users := newReferralUserRepository()
	referrer := users.mustCreate("+919876540017", "Referrer", "REFER103", nil)
	referrals := newReferralRepository()
	transaction := referrals.mustTransaction(referrer.ID, "Pending")
	gateway := NewMockReferralPayoutGateway()
	gateway.Fail = true
	service := NewReferralService(users, referrals, newReferralNotificationRepository(), gateway)

	result, err := service.ProcessPendingPayouts(ctx, 10)
	if err != nil {
		t.Fatalf("process payouts: %v", err)
	}
	if result.Failed != 1 || referrals.transactions[transaction.ID].Status != "Failed" {
		t.Fatalf("expected failed result, got result=%#v transaction=%#v", result, referrals.transactions[transaction.ID])
	}
}

type referralUserRepository struct {
	byID    map[string]*models.User
	byPhone map[string]*models.User
	byCode  map[string]*models.User
	nextID  int
}

func newReferralUserRepository() *referralUserRepository {
	return &referralUserRepository{
		byID:    map[string]*models.User{},
		byPhone: map[string]*models.User{},
		byCode:  map[string]*models.User{},
	}
}

func (r *referralUserRepository) mustCreate(phone, name, code string, referredByCode *string) *models.User {
	user, err := r.CreateUser(context.Background(), &models.User{
		PhoneNumber:        phone,
		FullName:           name,
		LanguagePreference: "en",
		VerificationTier:   models.VerificationTierHigh,
		ReferralCode:       code,
		ReferredByCode:     referredByCode,
		IsActive:           true,
	})
	if err != nil {
		panic(err)
	}
	return user
}

func (r *referralUserRepository) CreateUser(ctx context.Context, user *models.User) (*models.User, error) {
	r.nextID++
	copyUser := *user
	copyUser.ID = fmt.Sprintf("user-%d", r.nextID)
	copyUser.CreatedAt = time.Now()
	copyUser.UpdatedAt = copyUser.CreatedAt
	r.byID[copyUser.ID] = &copyUser
	r.byPhone[copyUser.PhoneNumber] = &copyUser
	r.byCode[copyUser.ReferralCode] = &copyUser
	return &copyUser, nil
}

func (r *referralUserRepository) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	user, ok := r.byID[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	copyUser := *user
	return &copyUser, nil
}

func (r *referralUserRepository) GetUserByPhone(ctx context.Context, phoneNumber string) (*models.User, error) {
	user, ok := r.byPhone[phoneNumber]
	if !ok {
		return nil, repository.ErrNotFound
	}
	copyUser := *user
	return &copyUser, nil
}

func (r *referralUserRepository) UpdateUserProfile(ctx context.Context, user *models.User) (*models.User, error) {
	copyUser := *user
	r.byID[copyUser.ID] = &copyUser
	r.byPhone[copyUser.PhoneNumber] = &copyUser
	r.byCode[copyUser.ReferralCode] = &copyUser
	return &copyUser, nil
}

func (r *referralUserRepository) UpdateUserVerificationTier(ctx context.Context, id string, tier models.VerificationTier) (*models.User, error) {
	user, err := r.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}
	user.VerificationTier = tier
	return r.UpdateUserProfile(ctx, user)
}

func (r *referralUserRepository) GetUserByReferralCode(ctx context.Context, referralCode string) (*models.User, error) {
	user, ok := r.byCode[referralCode]
	if !ok {
		return nil, repository.ErrNotFound
	}
	copyUser := *user
	return &copyUser, nil
}

type referralRepository struct {
	referrals    map[string]*models.Referral
	byReferredID map[string]*models.Referral
	transactions map[string]*models.ReferralTransaction
	nextReferral int
	nextTx       int
}

func newReferralRepository() *referralRepository {
	return &referralRepository{
		referrals:    map[string]*models.Referral{},
		byReferredID: map[string]*models.Referral{},
		transactions: map[string]*models.ReferralTransaction{},
	}
}

func (r *referralRepository) CreateReferral(ctx context.Context, referral *models.Referral) (*models.Referral, error) {
	r.nextReferral++
	copyReferral := *referral
	copyReferral.ID = fmt.Sprintf("referral-%d", r.nextReferral)
	copyReferral.CreatedAt = time.Now()
	r.referrals[copyReferral.ID] = &copyReferral
	if copyReferral.ReferredUserID != nil {
		r.byReferredID[*copyReferral.ReferredUserID] = &copyReferral
	}
	return &copyReferral, nil
}

func (r *referralRepository) GetReferralByReferredUserID(ctx context.Context, referredUserID string) (*models.Referral, error) {
	referral, ok := r.byReferredID[referredUserID]
	if !ok {
		return nil, repository.ErrNotFound
	}
	copyReferral := *referral
	return &copyReferral, nil
}

func (r *referralRepository) ListReferralsByReferrer(ctx context.Context, referrerUserID string, limit, offset int) ([]models.Referral, error) {
	var referrals []models.Referral
	for _, referral := range r.referrals {
		if referral.ReferrerUserID == referrerUserID {
			referrals = append(referrals, *referral)
		}
	}
	return referrals, nil
}

func (r *referralRepository) ListReferralTransactionsByUser(ctx context.Context, userID string, limit, offset int) ([]models.ReferralTransaction, error) {
	var transactions []models.ReferralTransaction
	for _, transaction := range r.transactions {
		if transaction.UserID == userID {
			transactions = append(transactions, *transaction)
		}
	}
	return transactions, nil
}

func (r *referralRepository) ListReferralTransactions(ctx context.Context, filters repository.ReferralTransactionFilters) ([]models.ReferralTransaction, error) {
	var transactions []models.ReferralTransaction
	for _, transaction := range r.transactions {
		if filters.Status != nil && transaction.Status != *filters.Status {
			continue
		}
		transactions = append(transactions, *transaction)
	}
	return transactions, nil
}

func (r *referralRepository) MarkReferralConverted(ctx context.Context, id string) (*models.Referral, error) {
	referral, ok := r.referrals[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	now := time.Now()
	referral.ConvertedAt = &now
	copyReferral := *referral
	return &copyReferral, nil
}

func (r *referralRepository) CreateReferralTransaction(ctx context.Context, transaction *models.ReferralTransaction) (*models.ReferralTransaction, error) {
	r.nextTx++
	copyTx := *transaction
	copyTx.ID = fmt.Sprintf("tx-%d", r.nextTx)
	copyTx.CreatedAt = time.Now()
	r.transactions[copyTx.ID] = &copyTx
	return &copyTx, nil
}

func (r *referralRepository) ClaimPendingReferralTransactions(ctx context.Context, limit int) ([]models.ReferralTransaction, error) {
	var claimed []models.ReferralTransaction
	for _, transaction := range r.transactions {
		if transaction.Status == "Pending" {
			transaction.Status = "Processing"
			claimed = append(claimed, *transaction)
		}
	}
	return claimed, nil
}

func (r *referralRepository) MarkReferralTransactionPaid(ctx context.Context, id, externalReference string) (*models.ReferralTransaction, error) {
	transaction, ok := r.transactions[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	transaction.Status = "Paid"
	transaction.ExternalReference = &externalReference
	now := time.Now()
	transaction.PaidAt = &now
	return transaction, nil
}

func (r *referralRepository) MarkReferralTransactionFailed(ctx context.Context, id, reason string) (*models.ReferralTransaction, error) {
	transaction, ok := r.transactions[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	transaction.Status = "Failed"
	transaction.ExternalReference = &reason
	return transaction, nil
}

func (r *referralRepository) mustTransaction(userID, status string) *models.ReferralTransaction {
	transaction, err := r.CreateReferralTransaction(context.Background(), &models.ReferralTransaction{
		ReferralID:  "referral-1",
		UserID:      userID,
		AmountPaise: referralCashbackAmountPaise,
		Currency:    "INR",
		Status:      status,
	})
	if err != nil {
		panic(err)
	}
	return transaction
}

type referralNotificationRepository struct {
	events []models.NotificationEvent
}

func newReferralNotificationRepository() *referralNotificationRepository {
	return &referralNotificationRepository{}
}

func (r *referralNotificationRepository) CreateNotificationEvent(ctx context.Context, event *models.NotificationEvent) (*models.NotificationEvent, error) {
	event.ID = fmt.Sprintf("notification-%d", len(r.events)+1)
	r.events = append(r.events, *event)
	return event, nil
}

func (r *referralNotificationRepository) ClaimPendingNotificationEvents(ctx context.Context, limit int) ([]models.NotificationEvent, error) {
	return nil, nil
}

func (r *referralNotificationRepository) ListNotificationEvents(ctx context.Context, filters repository.NotificationEventFilters) ([]models.NotificationEvent, error) {
	return nil, nil
}

func (r *referralNotificationRepository) MarkNotificationEventSent(ctx context.Context, id string) (*models.NotificationEvent, error) {
	return nil, nil
}

func (r *referralNotificationRepository) MarkNotificationEventFailed(ctx context.Context, id string, reason string) (*models.NotificationEvent, error) {
	return nil, nil
}
