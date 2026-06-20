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
	return scanEmployer(r.db.QueryRow(ctx, `
		INSERT INTO employers (
			company_name,
			contact_name,
			email,
			phone_number,
			city,
			state,
			is_verified
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING `+employerColumns,
		employer.CompanyName,
		employer.ContactName,
		employer.Email,
		nullableString(employer.PhoneNumber),
		nullableString(employer.City),
		nullableString(employer.State),
		employer.IsVerified,
	))
}

func (r *PostgresEmployerRepository) GetEmployerByID(ctx context.Context, id string) (*models.Employer, error) {
	return scanEmployer(r.db.QueryRow(ctx, `SELECT `+employerColumns+` FROM employers WHERE id = $1`, id))
}

func (r *PostgresEmployerRepository) GetEmployerByEmail(ctx context.Context, email string) (*models.Employer, error) {
	return scanEmployer(r.db.QueryRow(ctx, `SELECT `+employerColumns+` FROM employers WHERE email = $1`, email))
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
