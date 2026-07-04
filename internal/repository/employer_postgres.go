package repository

import (
	"context"
	"database/sql"

	"bluecollarjob/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

const employerColumns = `
	id,
	company_name,
	contact_name,
	email,
	password_hash,
	phone_number,
	city,
	state,
	is_verified,
	created_at,
	updated_at`

type PostgresEmployerRepository struct {
	db queryer
}

func NewPostgresEmployerRepository(db *pgxpool.Pool) *PostgresEmployerRepository {
	return &PostgresEmployerRepository{db: db}
}

func (r *PostgresEmployerRepository) CreateEmployer(ctx context.Context, employer *models.Employer) (*models.Employer, error) {
	employerRecord, err := scanEmployer(r.db.QueryRow(ctx, `
		INSERT INTO employers (
			company_name,
			contact_name,
			email,
			password_hash,
			phone_number,
			city,
			state,
			is_verified
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING `+employerColumns,
		employer.CompanyName,
		employer.ContactName,
		employer.Email,
		employer.PasswordHash,
		nullableString(employer.PhoneNumber),
		nullableString(employer.City),
		nullableString(employer.State),
		employer.IsVerified,
	))
	if err != nil {
		return nil, mapPostgresError(err)
	}
	return employerRecord, nil
}

func (r *PostgresEmployerRepository) GetEmployerByID(ctx context.Context, id string) (*models.Employer, error) {
	return scanEmployer(r.db.QueryRow(ctx, `SELECT `+employerColumns+` FROM employers WHERE id = $1`, id))
}

func (r *PostgresEmployerRepository) GetEmployerByEmail(ctx context.Context, email string) (*models.Employer, error) {
	return scanEmployer(r.db.QueryRow(ctx, `SELECT `+employerColumns+` FROM employers WHERE email = $1`, email))
}

func (r *PostgresEmployerRepository) UpdateEmployerProfile(ctx context.Context, employer *models.Employer) (*models.Employer, error) {
	return scanEmployer(r.db.QueryRow(ctx, `
		UPDATE employers
		SET company_name = $2,
			contact_name = $3,
			phone_number = $4,
			city = $5,
			state = $6
		WHERE id = $1
		RETURNING `+employerColumns,
		employer.ID,
		employer.CompanyName,
		employer.ContactName,
		nullableString(employer.PhoneNumber),
		nullableString(employer.City),
		nullableString(employer.State),
	))
}

func (r *PostgresEmployerRepository) GetActiveSubscriptionTier(ctx context.Context, employerID string) (models.SubscriptionTier, error) {
	var tier models.SubscriptionTier
	err := r.db.QueryRow(ctx, `
		SELECT tier
		FROM subscriptions
		WHERE employer_id = $1
			AND is_active = TRUE
			AND (ends_at IS NULL OR ends_at > NOW())
		ORDER BY starts_at DESC
		LIMIT 1`,
		employerID,
	).Scan(&tier)
	if err != nil {
		return "", mapNotFound(err)
	}
	return tier, nil
}

func scanEmployer(row interface{ Scan(dest ...any) error }) (*models.Employer, error) {
	var employer models.Employer
	var phoneNumber sql.NullString
	var city sql.NullString
	var state sql.NullString

	err := row.Scan(
		&employer.ID,
		&employer.CompanyName,
		&employer.ContactName,
		&employer.Email,
		&employer.PasswordHash,
		&phoneNumber,
		&city,
		&state,
		&employer.IsVerified,
		&employer.CreatedAt,
		&employer.UpdatedAt,
	)
	if err != nil {
		return nil, mapNotFound(err)
	}

	employer.PhoneNumber = stringPtr(phoneNumber)
	employer.City = stringPtr(city)
	employer.State = stringPtr(state)

	return &employer, nil
}
