package repository

import (
	"context"
	"database/sql"
	"strings"

	"bluecollarjob/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const referralColumns = `
	id,
	referrer_user_id,
	referred_user_id,
	referral_code,
	created_at,
	converted_at`

const referralTransactionColumns = `
	id,
	referral_id,
	user_id,
	amount_paise,
	currency,
	status,
	external_reference,
	created_at,
	paid_at`

type PostgresReferralRepository struct {
	db queryer
}

func NewPostgresReferralRepository(db *pgxpool.Pool) *PostgresReferralRepository {
	return &PostgresReferralRepository{db: db}
}

func (r *PostgresReferralRepository) CreateReferral(ctx context.Context, referral *models.Referral) (*models.Referral, error) {
	return scanReferral(r.db.QueryRow(ctx, `
		INSERT INTO referrals (referrer_user_id, referred_user_id, referral_code)
		VALUES ($1, $2, $3)
		RETURNING `+referralColumns,
		referral.ReferrerUserID,
		nullableString(referral.ReferredUserID),
		referral.ReferralCode,
	))
}

func (r *PostgresReferralRepository) GetReferralByReferredUserID(ctx context.Context, referredUserID string) (*models.Referral, error) {
	return scanReferral(r.db.QueryRow(ctx, `SELECT `+referralColumns+` FROM referrals WHERE referred_user_id = $1`, referredUserID))
}

func (r *PostgresReferralRepository) ListReferralsByReferrer(ctx context.Context, referrerUserID string, limit, offset int) ([]models.Referral, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+referralColumns+`
		FROM referrals
		WHERE referrer_user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`,
		referrerUserID,
		normalizeLimit(limit),
		normalizeOffset(offset),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanReferrals(rows)
}

func (r *PostgresReferralRepository) ListReferralTransactionsByUser(ctx context.Context, userID string, limit, offset int) ([]models.ReferralTransaction, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+referralTransactionColumns+`
		FROM referral_transactions
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`,
		userID,
		normalizeLimit(limit),
		normalizeOffset(offset),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanReferralTransactions(rows)
}

func (r *PostgresReferralRepository) MarkReferralConverted(ctx context.Context, id string) (*models.Referral, error) {
	return scanReferral(r.db.QueryRow(ctx, `
		UPDATE referrals
		SET converted_at = COALESCE(converted_at, NOW())
		WHERE id = $1
		RETURNING `+referralColumns,
		id,
	))
}

func (r *PostgresReferralRepository) CreateReferralTransaction(ctx context.Context, transaction *models.ReferralTransaction) (*models.ReferralTransaction, error) {
	return scanReferralTransaction(r.db.QueryRow(ctx, `
		INSERT INTO referral_transactions (
			referral_id,
			user_id,
			amount_paise,
			currency,
			status,
			external_reference
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING `+referralTransactionColumns,
		transaction.ReferralID,
		transaction.UserID,
		transaction.AmountPaise,
		transaction.Currency,
		transaction.Status,
		nullableString(transaction.ExternalReference),
	))
}

func (r *PostgresReferralRepository) ClaimPendingReferralTransactions(ctx context.Context, limit int) ([]models.ReferralTransaction, error) {
	rows, err := r.db.Query(ctx, `
		WITH claimed AS (
			SELECT id
			FROM referral_transactions
			WHERE status = 'Pending'
			ORDER BY created_at ASC
			LIMIT $1
			FOR UPDATE SKIP LOCKED
		)
		UPDATE referral_transactions t
		SET status = 'Processing'
		FROM claimed
		WHERE t.id = claimed.id
		RETURNING
			t.id,
			t.referral_id,
			t.user_id,
			t.amount_paise,
			t.currency,
			t.status,
			t.external_reference,
			t.created_at,
			t.paid_at`,
		normalizeLimit(limit),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanReferralTransactions(rows)
}

func (r *PostgresReferralRepository) MarkReferralTransactionPaid(ctx context.Context, id, externalReference string) (*models.ReferralTransaction, error) {
	return scanReferralTransaction(r.db.QueryRow(ctx, `
		UPDATE referral_transactions
		SET status = 'Paid',
			external_reference = $2,
			paid_at = NOW()
		WHERE id = $1
		RETURNING `+referralTransactionColumns,
		id,
		externalReference,
	))
}

func (r *PostgresReferralRepository) MarkReferralTransactionFailed(ctx context.Context, id, reason string) (*models.ReferralTransaction, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "mock payout failed"
	}
	return scanReferralTransaction(r.db.QueryRow(ctx, `
		UPDATE referral_transactions
		SET status = 'Failed',
			external_reference = $2
		WHERE id = $1
		RETURNING `+referralTransactionColumns,
		id,
		reason,
	))
}

func scanReferrals(rows pgx.Rows) ([]models.Referral, error) {
	var referrals []models.Referral
	for rows.Next() {
		referral, err := scanReferral(rows)
		if err != nil {
			return nil, err
		}
		referrals = append(referrals, *referral)
	}
	return referrals, rows.Err()
}

func scanReferral(row interface{ Scan(dest ...any) error }) (*models.Referral, error) {
	var referral models.Referral
	var referredUserID sql.NullString
	var convertedAt sql.NullTime
	err := row.Scan(
		&referral.ID,
		&referral.ReferrerUserID,
		&referredUserID,
		&referral.ReferralCode,
		&referral.CreatedAt,
		&convertedAt,
	)
	if err != nil {
		return nil, mapNotFound(err)
	}
	referral.ReferredUserID = stringPtr(referredUserID)
	referral.ConvertedAt = timePtr(convertedAt)
	return &referral, nil
}

func scanReferralTransactions(rows pgx.Rows) ([]models.ReferralTransaction, error) {
	var transactions []models.ReferralTransaction
	for rows.Next() {
		transaction, err := scanReferralTransaction(rows)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, *transaction)
	}
	return transactions, rows.Err()
}

func scanReferralTransaction(row interface{ Scan(dest ...any) error }) (*models.ReferralTransaction, error) {
	var transaction models.ReferralTransaction
	var externalReference sql.NullString
	var paidAt sql.NullTime
	err := row.Scan(
		&transaction.ID,
		&transaction.ReferralID,
		&transaction.UserID,
		&transaction.AmountPaise,
		&transaction.Currency,
		&transaction.Status,
		&externalReference,
		&transaction.CreatedAt,
		&paidAt,
	)
	if err != nil {
		return nil, mapNotFound(err)
	}
	transaction.ExternalReference = stringPtr(externalReference)
	transaction.PaidAt = timePtr(paidAt)
	return &transaction, nil
}
