package service

import (
	"context"
	"testing"
	"time"

	"bluecollarjob/internal/models"
	"bluecollarjob/internal/repository"
)

func TestIdentityVerificationServiceAadhaarOTPVerifiedSetsLowRisk(t *testing.T) {
	ctx := context.Background()
	users := newFakeUserRepository()
	verifications := newFakeIdentityVerificationRepository()
	gateway := &fakeAadhaarGateway{transactionID: "txn-test-123", verified: true}

	user := users.seedUser(&models.User{
		ID:                 "user-1",
		PhoneNumber:        "+919876543210",
		FullName:           "Test Worker",
		LanguagePreference: "en",
		VerificationTier:   models.VerificationTierHigh,
		ReferralCode:       "TEST100",
		IsActive:           true,
	})

	service := NewIdentityVerificationService(users, verifications, gateway)
	verification, err := service.StartAadhaarOTP(ctx, user.ID, "123456789012", true)
	if err != nil {
		t.Fatalf("start aadhaar otp: %v", err)
	}

	if verification.AadhaarLast4 == nil || *verification.AadhaarLast4 != "9012" {
		t.Fatalf("expected aadhaar last4 9012, got %#v", verification.AadhaarLast4)
	}
	if verification.AadhaarMasked == nil || *verification.AadhaarMasked != "XXXX-XXXX-9012" {
		t.Fatalf("expected masked aadhaar, got %#v", verification.AadhaarMasked)
	}
	if verification.AadhaarHash == nil || *verification.AadhaarHash == "123456789012" {
		t.Fatal("raw aadhaar must not be stored as hash")
	}

	verification, err = service.VerifyAadhaarOTP(ctx, user.ID, "txn-test-123", "123456")
	if err != nil {
		t.Fatalf("verify aadhaar otp: %v", err)
	}
	if verification.Status != models.IdentityVerificationStatusVerified {
		t.Fatalf("expected verified status, got %s", verification.Status)
	}
	if users.users[user.ID].VerificationTier != models.VerificationTierLow {
		t.Fatalf("expected low risk tier, got %s", users.users[user.ID].VerificationTier)
	}
}

func TestIdentityVerificationServiceDocumentUploadSetsMediumRisk(t *testing.T) {
	ctx := context.Background()
	users := newFakeUserRepository()
	verifications := newFakeIdentityVerificationRepository()
	user := users.seedUser(&models.User{
		ID:                 "user-2",
		PhoneNumber:        "+919876543211",
		FullName:           "Document Worker",
		LanguagePreference: "en",
		VerificationTier:   models.VerificationTierHigh,
		ReferralCode:       "DOC100",
		IsActive:           true,
	})

	service := NewIdentityVerificationService(users, verifications, &fakeAadhaarGateway{verified: true})
	verification, err := service.MarkDocumentUploaded(ctx, user.ID, "s3://bucket/document.jpg")
	if err != nil {
		t.Fatalf("mark document uploaded: %v", err)
	}

	if verification.Status != models.IdentityVerificationStatusDocumentUploaded {
		t.Fatalf("expected document uploaded status, got %s", verification.Status)
	}
	if verification.DocumentRef == nil {
		t.Fatal("expected document reference to be stored")
	}
	if users.users[user.ID].VerificationTier != models.VerificationTierMedium {
		t.Fatalf("expected medium risk tier, got %s", users.users[user.ID].VerificationTier)
	}
}

func TestIdentityVerificationServiceSkipSetsHighRisk(t *testing.T) {
	ctx := context.Background()
	users := newFakeUserRepository()
	verifications := newFakeIdentityVerificationRepository()
	user := users.seedUser(&models.User{
		ID:                 "user-3",
		PhoneNumber:        "+919876543212",
		FullName:           "Skipped Worker",
		LanguagePreference: "en",
		VerificationTier:   models.VerificationTierLow,
		ReferralCode:       "SKIP100",
		IsActive:           true,
	})

	service := NewIdentityVerificationService(users, verifications, &fakeAadhaarGateway{verified: true})
	verification, err := service.MarkSkipped(ctx, user.ID, "skipped in test")
	if err != nil {
		t.Fatalf("mark skipped: %v", err)
	}

	if verification.Status != models.IdentityVerificationStatusSkipped {
		t.Fatalf("expected skipped status, got %s", verification.Status)
	}
	if users.users[user.ID].VerificationTier != models.VerificationTierHigh {
		t.Fatalf("expected high risk tier, got %s", users.users[user.ID].VerificationTier)
	}
}

type fakeAadhaarGateway struct {
	transactionID string
	verified      bool
}

func (g *fakeAadhaarGateway) SendOTP(ctx context.Context, request AadhaarOTPRequest) (AadhaarOTPResponse, error) {
	transactionID := g.transactionID
	if transactionID == "" {
		transactionID = "txn-test"
	}
	return AadhaarOTPResponse{TransactionID: transactionID, MobileLinked: true}, nil
}

func (g *fakeAadhaarGateway) VerifyOTP(ctx context.Context, request AadhaarOTPVerificationRequest) (AadhaarOTPVerificationResponse, error) {
	return AadhaarOTPVerificationResponse{Verified: g.verified}, nil
}

type fakeUserRepository struct {
	users map[string]*models.User
}

func newFakeUserRepository() *fakeUserRepository {
	return &fakeUserRepository{users: map[string]*models.User{}}
}

func (r *fakeUserRepository) seedUser(user *models.User) *models.User {
	r.users[user.ID] = user
	return user
}

func (r *fakeUserRepository) CreateUser(ctx context.Context, user *models.User) (*models.User, error) {
	r.users[user.ID] = user
	return user, nil
}

func (r *fakeUserRepository) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	user, ok := r.users[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return user, nil
}

func (r *fakeUserRepository) GetUserByPhone(ctx context.Context, phoneNumber string) (*models.User, error) {
	for _, user := range r.users {
		if user.PhoneNumber == phoneNumber {
			return user, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (r *fakeUserRepository) UpdateUserProfile(ctx context.Context, user *models.User) (*models.User, error) {
	r.users[user.ID] = user
	return user, nil
}

func (r *fakeUserRepository) UpdateUserVerificationTier(ctx context.Context, id string, tier models.VerificationTier) (*models.User, error) {
	user, ok := r.users[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	user.VerificationTier = tier
	return user, nil
}

func (r *fakeUserRepository) GetUserByReferralCode(ctx context.Context, referralCode string) (*models.User, error) {
	for _, user := range r.users {
		if user.ReferralCode == referralCode {
			return user, nil
		}
	}
	return nil, repository.ErrNotFound
}

type fakeIdentityVerificationRepository struct {
	records map[string]*models.WorkerIdentityVerification
	latest  map[string]string
	nextID  int
}

func newFakeIdentityVerificationRepository() *fakeIdentityVerificationRepository {
	return &fakeIdentityVerificationRepository{
		records: map[string]*models.WorkerIdentityVerification{},
		latest:  map[string]string{},
	}
}

func (r *fakeIdentityVerificationRepository) CreateVerificationRecord(ctx context.Context, verification *models.WorkerIdentityVerification) (*models.WorkerIdentityVerification, error) {
	r.nextID++
	verification.ID = "verification-test"
	if r.nextID > 1 {
		verification.ID = verification.ID + "-" + string(rune('0'+r.nextID))
	}
	verification.CreatedAt = time.Now().UTC()
	verification.UpdatedAt = verification.CreatedAt
	r.records[verification.ID] = verification
	r.latest[verification.UserID] = verification.ID
	return verification, nil
}

func (r *fakeIdentityVerificationRepository) GetLatestVerificationByUserID(ctx context.Context, userID string) (*models.WorkerIdentityVerification, error) {
	id, ok := r.latest[userID]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return r.records[id], nil
}

func (r *fakeIdentityVerificationRepository) MarkOTPVerificationPending(ctx context.Context, id string, aadhaarLast4, aadhaarMasked, aadhaarHash, aadhaarReferenceKey string, consentGivenAt time.Time) (*models.WorkerIdentityVerification, error) {
	record, ok := r.records[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	record.Method = models.IdentityVerificationMethodAadhaarOTP
	record.Status = models.IdentityVerificationStatusOTPSent
	record.AadhaarLast4 = &aadhaarLast4
	record.AadhaarMasked = &aadhaarMasked
	record.AadhaarHash = &aadhaarHash
	record.AadhaarReferenceKey = &aadhaarReferenceKey
	record.ConsentGiven = true
	record.ConsentGivenAt = &consentGivenAt
	return record, nil
}

func (r *fakeIdentityVerificationRepository) MarkVerified(ctx context.Context, id string) (*models.WorkerIdentityVerification, error) {
	record, ok := r.records[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	now := time.Now().UTC()
	record.Status = models.IdentityVerificationStatusVerified
	record.VerifiedAt = &now
	return record, nil
}

func (r *fakeIdentityVerificationRepository) MarkDocumentUploaded(ctx context.Context, id string, documentRef string) (*models.WorkerIdentityVerification, error) {
	record, ok := r.records[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	record.Method = models.IdentityVerificationMethodDocumentUpload
	record.Status = models.IdentityVerificationStatusDocumentUploaded
	record.DocumentRef = &documentRef
	return record, nil
}

func (r *fakeIdentityVerificationRepository) MarkSkipped(ctx context.Context, id string, reason string) (*models.WorkerIdentityVerification, error) {
	record, ok := r.records[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	record.Method = models.IdentityVerificationMethodSkipped
	record.Status = models.IdentityVerificationStatusSkipped
	record.FailedReason = &reason
	return record, nil
}

func (r *fakeIdentityVerificationRepository) MarkFailed(ctx context.Context, id string, reason string) (*models.WorkerIdentityVerification, error) {
	record, ok := r.records[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	record.Status = models.IdentityVerificationStatusFailed
	record.FailedReason = &reason
	return record, nil
}
