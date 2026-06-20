package repository

import (
	"context"
	"database/sql"

	"bluecollarjob/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

const userColumns = `
	id,
	phone_number,
	full_name,
	language_preference,
	target_role,
	preferred_zone,
	verification_tier,
	referral_code,
	referred_by_code,
	is_active,
	created_at,
	updated_at`

type PostgresUserRepository struct {
	db queryer
}

func NewPostgresUserRepository(db *pgxpool.Pool) *PostgresUserRepository {
	return &PostgresUserRepository{db: db}
}

func (r *PostgresUserRepository) CreateUser(ctx context.Context, user *models.User) (*models.User, error) {
	row := r.db.QueryRow(ctx, `
		INSERT INTO users (
			phone_number,
			full_name,
			language_preference,
			target_role,
			preferred_zone,
			verification_tier,
			referral_code,
			referred_by_code,
			is_active
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, COALESCE($9, TRUE))
		RETURNING `+userColumns,
		user.PhoneNumber,
		user.FullName,
		user.LanguagePreference,
		nullableString(user.TargetRole),
		nullableString(user.PreferredZone),
		user.VerificationTier,
		user.ReferralCode,
		nullableString(user.ReferredByCode),
		user.IsActive,
	)

	return scanUser(row)
}

func (r *PostgresUserRepository) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	return scanUser(r.db.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE id = $1`, id))
}

func (r *PostgresUserRepository) GetUserByPhone(ctx context.Context, phoneNumber string) (*models.User, error) {
	return scanUser(r.db.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE phone_number = $1`, phoneNumber))
}

func (r *PostgresUserRepository) UpdateUserVerificationTier(ctx context.Context, id string, tier models.VerificationTier) (*models.User, error) {
	return scanUser(r.db.QueryRow(ctx, `
		UPDATE users
		SET verification_tier = $2
		WHERE id = $1
		RETURNING `+userColumns,
		id,
		tier,
	))
}

func (r *PostgresUserRepository) GetUserByReferralCode(ctx context.Context, referralCode string) (*models.User, error) {
	return scanUser(r.db.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE referral_code = $1`, referralCode))
}

func scanUser(row interface{ Scan(dest ...any) error }) (*models.User, error) {
	var user models.User
	var targetRole sql.NullString
	var preferredZone sql.NullString
	var referredByCode sql.NullString

	err := row.Scan(
		&user.ID,
		&user.PhoneNumber,
		&user.FullName,
		&user.LanguagePreference,
		&targetRole,
		&preferredZone,
		&user.VerificationTier,
		&user.ReferralCode,
		&referredByCode,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, mapNotFound(err)
	}

	user.TargetRole = stringPtr(targetRole)
	user.PreferredZone = stringPtr(preferredZone)
	user.ReferredByCode = stringPtr(referredByCode)

	return &user, nil
}
