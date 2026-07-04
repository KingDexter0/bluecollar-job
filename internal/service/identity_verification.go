package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"bluecollarjob/internal/models"
	"bluecollarjob/internal/repository"
)

type IdentityVerificationService interface {
	StartAadhaarOTP(ctx context.Context, userID, aadhaarNumber string, consentGiven bool) (*models.WorkerIdentityVerification, error)
	VerifyAadhaarOTP(ctx context.Context, userID, transactionID, otp string) (*models.WorkerIdentityVerification, error)
	MarkDocumentUploaded(ctx context.Context, userID, documentRef string) (*models.WorkerIdentityVerification, error)
	MarkSkipped(ctx context.Context, userID, reason string) (*models.WorkerIdentityVerification, error)
	GetLatest(ctx context.Context, userID string) (*models.WorkerIdentityVerification, error)
}

type identityVerificationService struct {
	users         repository.UserRepository
	verifications repository.IdentityVerificationRepository
	gateway       AadhaarGateway
	referrals     ReferralService
	now           func() time.Time
}

func NewIdentityVerificationService(users repository.UserRepository, verifications repository.IdentityVerificationRepository, gateway AadhaarGateway, referrals ...ReferralService) IdentityVerificationService {
	var referralService ReferralService
	if len(referrals) > 0 {
		referralService = referrals[0]
	}
	return &identityVerificationService{
		users:         users,
		verifications: verifications,
		gateway:       gateway,
		referrals:     referralService,
		now:           func() time.Time { return time.Now().UTC() },
	}
}

func (s *identityVerificationService) StartAadhaarOTP(ctx context.Context, userID, aadhaarNumber string, consentGiven bool) (*models.WorkerIdentityVerification, error) {
	if !consentGiven {
		return nil, fmt.Errorf("%w: aadhaar verification consent is required", ErrInvalidInput)
	}

	user, err := s.users.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	otpResponse, err := s.gateway.SendOTP(ctx, AadhaarOTPRequest{
		AadhaarNumber: aadhaarNumber,
		PhoneNumber:   user.PhoneNumber,
	})
	if err != nil {
		return nil, err
	}
	if !otpResponse.MobileLinked {
		return nil, fmt.Errorf("%w: aadhaar linked mobile unavailable", ErrInvalidInput)
	}

	last4 := aadhaarNumber[len(aadhaarNumber)-4:]
	masked := "XXXX-XXXX-" + last4
	hash := hashAadhaar(aadhaarNumber)
	referenceKey := otpResponse.TransactionID
	consentAt := s.now()

	verification, err := s.verifications.CreateVerificationRecord(ctx, &models.WorkerIdentityVerification{
		UserID:              user.ID,
		Method:              models.IdentityVerificationMethodAadhaarOTP,
		Status:              models.IdentityVerificationStatusOTPSent,
		AadhaarLast4:        &last4,
		AadhaarMasked:       &masked,
		AadhaarHash:         &hash,
		AadhaarReferenceKey: &referenceKey,
		ConsentGiven:        true,
		ConsentGivenAt:      &consentAt,
	})
	if err != nil {
		return nil, err
	}

	return verification, nil
}

func (s *identityVerificationService) VerifyAadhaarOTP(ctx context.Context, userID, transactionID, otp string) (*models.WorkerIdentityVerification, error) {
	verification, err := s.verifications.GetLatestVerificationByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if verification.Method != models.IdentityVerificationMethodAadhaarOTP || verification.Status != models.IdentityVerificationStatusOTPSent {
		return nil, fmt.Errorf("%w: no pending aadhaar otp verification", ErrInvalidInput)
	}

	result, err := s.gateway.VerifyOTP(ctx, AadhaarOTPVerificationRequest{
		TransactionID: transactionID,
		OTP:           otp,
	})
	if err != nil {
		return nil, err
	}
	if !result.Verified {
		return s.verifications.MarkFailed(ctx, verification.ID, "Aadhaar OTP verification failed")
	}

	verification, err = s.verifications.MarkVerified(ctx, verification.ID)
	if err != nil {
		return nil, err
	}
	if _, err := s.users.UpdateUserVerificationTier(ctx, userID, models.VerificationTierLow); err != nil {
		return nil, err
	}
	if s.referrals != nil {
		if _, err := s.referrals.CompleteOnboarding(ctx, userID); err != nil {
			return nil, err
		}
	}

	return verification, nil
}

func (s *identityVerificationService) MarkDocumentUploaded(ctx context.Context, userID, documentRef string) (*models.WorkerIdentityVerification, error) {
	now := s.now()
	latest, err := s.verifications.GetLatestVerificationByUserID(ctx, userID)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}
	if latest != nil && isActiveVerificationStatus(latest.Status) {
		verification, err := s.verifications.MarkDocumentUploaded(ctx, latest.ID, documentRef)
		if err != nil {
			return nil, err
		}
		if _, err := s.users.UpdateUserVerificationTier(ctx, userID, models.VerificationTierMedium); err != nil {
			return nil, err
		}
		if s.referrals != nil {
			if _, err := s.referrals.CompleteOnboarding(ctx, userID); err != nil {
				return nil, err
			}
		}
		return verification, nil
	}

	verification, err := s.verifications.CreateVerificationRecord(ctx, &models.WorkerIdentityVerification{
		UserID:         userID,
		Method:         models.IdentityVerificationMethodDocumentUpload,
		Status:         models.IdentityVerificationStatusDocumentUploaded,
		DocumentRef:    &documentRef,
		ConsentGiven:   true,
		ConsentGivenAt: &now,
		FailedReason:   stringPointer("Aadhaar OTP unavailable or incomplete; document uploaded for review."),
	})
	if err != nil {
		return nil, err
	}
	if _, err := s.users.UpdateUserVerificationTier(ctx, userID, models.VerificationTierMedium); err != nil {
		return nil, err
	}
	if s.referrals != nil {
		if _, err := s.referrals.CompleteOnboarding(ctx, userID); err != nil {
			return nil, err
		}
	}

	return verification, nil
}

func (s *identityVerificationService) MarkSkipped(ctx context.Context, userID, reason string) (*models.WorkerIdentityVerification, error) {
	if reason == "" {
		reason = "Worker skipped identity verification during onboarding."
	}

	latest, err := s.verifications.GetLatestVerificationByUserID(ctx, userID)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}
	if latest != nil && isActiveVerificationStatus(latest.Status) {
		verification, err := s.verifications.MarkSkipped(ctx, latest.ID, reason)
		if err != nil {
			return nil, err
		}
		if _, err := s.users.UpdateUserVerificationTier(ctx, userID, models.VerificationTierHigh); err != nil {
			return nil, err
		}
		if s.referrals != nil {
			if _, err := s.referrals.CompleteOnboarding(ctx, userID); err != nil {
				return nil, err
			}
		}
		return verification, nil
	}

	verification, err := s.verifications.CreateVerificationRecord(ctx, &models.WorkerIdentityVerification{
		UserID:       userID,
		Method:       models.IdentityVerificationMethodSkipped,
		Status:       models.IdentityVerificationStatusSkipped,
		ConsentGiven: false,
		FailedReason: &reason,
	})
	if err != nil {
		return nil, err
	}
	if _, err := s.users.UpdateUserVerificationTier(ctx, userID, models.VerificationTierHigh); err != nil {
		return nil, err
	}
	if s.referrals != nil {
		if _, err := s.referrals.CompleteOnboarding(ctx, userID); err != nil {
			return nil, err
		}
	}

	return verification, nil
}

func (s *identityVerificationService) GetLatest(ctx context.Context, userID string) (*models.WorkerIdentityVerification, error) {
	return s.verifications.GetLatestVerificationByUserID(ctx, userID)
}

func hashAadhaar(aadhaarNumber string) string {
	sum := sha256.Sum256([]byte(aadhaarNumber))
	return hex.EncodeToString(sum[:])
}

func stringPointer(value string) *string {
	return &value
}

func isActiveVerificationStatus(status models.IdentityVerificationStatus) bool {
	return status == models.IdentityVerificationStatusPending ||
		status == models.IdentityVerificationStatusOTPSent ||
		status == models.IdentityVerificationStatusDocumentUploaded
}
