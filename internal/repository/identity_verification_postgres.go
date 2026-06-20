package repository

import (
	"context"
	"database/sql"
	"time"

	"bluecollarjob/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

const identityVerificationColumns = `
	id,
	user_id,
	method,
	status,
	aadhaar_last4,
	aadhaar_masked,
	aadhaar_hash,
	aadhaar_reference_key,
	document_ref,
	consent_given,
	consent_given_at,
	verified_at,
	failed_reason,
	created_at,
	updated_at`

type PostgresIdentityVerificationRepository struct {
	db queryer
}

func NewPostgresIdentityVerificationRepository(db *pgxpool.Pool) *PostgresIdentityVerificationRepository {
	return &PostgresIdentityVerificationRepository{db: db}
}

func (r *PostgresIdentityVerificationRepository) CreateVerificationRecord(ctx context.Context, verification *models.WorkerIdentityVerification) (*models.WorkerIdentityVerification, error) {
	return scanIdentityVerification(r.db.QueryRow(ctx, `
		INSERT INTO worker_identity_verifications (
			user_id,
			method,
			status,
			aadhaar_last4,
			aadhaar_masked,
			aadhaar_hash,
			aadhaar_reference_key,
			document_ref,
			consent_given,
			consent_given_at,
			verified_at,
			failed_reason
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING `+identityVerificationColumns,
		verification.UserID,
		verification.Method,
		verification.Status,
		nullableString(verification.AadhaarLast4),
		nullableString(verification.AadhaarMasked),
		nullableString(verification.AadhaarHash),
		nullableString(verification.AadhaarReferenceKey),
		nullableString(verification.DocumentRef),
		verification.ConsentGiven,
		nullableTime(verification.ConsentGivenAt),
		nullableTime(verification.VerifiedAt),
		nullableString(verification.FailedReason),
	))
}

func (r *PostgresIdentityVerificationRepository) GetLatestVerificationByUserID(ctx context.Context, userID string) (*models.WorkerIdentityVerification, error) {
	return scanIdentityVerification(r.db.QueryRow(ctx, `
		SELECT `+identityVerificationColumns+`
		FROM worker_identity_verifications
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT 1`,
		userID,
	))
}

func (r *PostgresIdentityVerificationRepository) MarkOTPVerificationPending(ctx context.Context, id string, aadhaarLast4, aadhaarMasked, aadhaarHash, aadhaarReferenceKey string, consentGivenAt time.Time) (*models.WorkerIdentityVerification, error) {
	return scanIdentityVerification(r.db.QueryRow(ctx, `
		UPDATE worker_identity_verifications
		SET method = 'Aadhaar_OTP',
			status = 'OTP_Sent',
			aadhaar_last4 = $2,
			aadhaar_masked = $3,
			aadhaar_hash = $4,
			aadhaar_reference_key = $5,
			document_ref = NULL,
			consent_given = TRUE,
			consent_given_at = $6,
			verified_at = NULL,
			failed_reason = NULL
		WHERE id = $1
		RETURNING `+identityVerificationColumns,
		id,
		aadhaarLast4,
		aadhaarMasked,
		aadhaarHash,
		aadhaarReferenceKey,
		consentGivenAt,
	))
}

func (r *PostgresIdentityVerificationRepository) MarkVerified(ctx context.Context, id string) (*models.WorkerIdentityVerification, error) {
	return scanIdentityVerification(r.db.QueryRow(ctx, `
		UPDATE worker_identity_verifications
		SET status = 'Verified',
			verified_at = NOW(),
			failed_reason = NULL
		WHERE id = $1
		RETURNING `+identityVerificationColumns,
		id,
	))
}

func (r *PostgresIdentityVerificationRepository) MarkDocumentUploaded(ctx context.Context, id string, documentRef string) (*models.WorkerIdentityVerification, error) {
	return scanIdentityVerification(r.db.QueryRow(ctx, `
		UPDATE worker_identity_verifications
		SET method = 'Document_Upload',
			status = 'Document_Uploaded',
			document_ref = $2,
			verified_at = NULL,
			failed_reason = NULL
		WHERE id = $1
		RETURNING `+identityVerificationColumns,
		id,
		documentRef,
	))
}

func (r *PostgresIdentityVerificationRepository) MarkSkipped(ctx context.Context, id string, reason string) (*models.WorkerIdentityVerification, error) {
	return scanIdentityVerification(r.db.QueryRow(ctx, `
		UPDATE worker_identity_verifications
		SET method = 'Skipped',
			status = 'Skipped',
			aadhaar_last4 = NULL,
			aadhaar_masked = NULL,
			aadhaar_hash = NULL,
			aadhaar_reference_key = NULL,
			document_ref = NULL,
			consent_given = FALSE,
			consent_given_at = NULL,
			verified_at = NULL,
			failed_reason = $2
		WHERE id = $1
		RETURNING `+identityVerificationColumns,
		id,
		reason,
	))
}

func (r *PostgresIdentityVerificationRepository) MarkFailed(ctx context.Context, id string, reason string) (*models.WorkerIdentityVerification, error) {
	return scanIdentityVerification(r.db.QueryRow(ctx, `
		UPDATE worker_identity_verifications
		SET status = 'Failed',
			verified_at = NULL,
			failed_reason = $2
		WHERE id = $1
		RETURNING `+identityVerificationColumns,
		id,
		reason,
	))
}

func scanIdentityVerification(row interface{ Scan(dest ...any) error }) (*models.WorkerIdentityVerification, error) {
	var verification models.WorkerIdentityVerification
	var aadhaarLast4 sql.NullString
	var aadhaarMasked sql.NullString
	var aadhaarHash sql.NullString
	var aadhaarReferenceKey sql.NullString
	var documentRef sql.NullString
	var consentGivenAt sql.NullTime
	var verifiedAt sql.NullTime
	var failedReason sql.NullString

	err := row.Scan(
		&verification.ID,
		&verification.UserID,
		&verification.Method,
		&verification.Status,
		&aadhaarLast4,
		&aadhaarMasked,
		&aadhaarHash,
		&aadhaarReferenceKey,
		&documentRef,
		&verification.ConsentGiven,
		&consentGivenAt,
		&verifiedAt,
		&failedReason,
		&verification.CreatedAt,
		&verification.UpdatedAt,
	)
	if err != nil {
		return nil, mapNotFound(err)
	}

	verification.AadhaarLast4 = stringPtr(aadhaarLast4)
	verification.AadhaarMasked = stringPtr(aadhaarMasked)
	verification.AadhaarHash = stringPtr(aadhaarHash)
	verification.AadhaarReferenceKey = stringPtr(aadhaarReferenceKey)
	verification.DocumentRef = stringPtr(documentRef)
	verification.ConsentGivenAt = timePtr(consentGivenAt)
	verification.VerifiedAt = timePtr(verifiedAt)
	verification.FailedReason = stringPtr(failedReason)

	return &verification, nil
}
