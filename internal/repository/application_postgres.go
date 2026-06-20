package repository

import (
	"context"

	"bluecollarjob/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const applicationColumns = `
	id,
	user_id,
	job_id,
	employer_id,
	status,
	source,
	applied_at,
	created_at,
	updated_at`

type PostgresApplicationRepository struct {
	db queryer
}

func NewPostgresApplicationRepository(db *pgxpool.Pool) *PostgresApplicationRepository {
	return &PostgresApplicationRepository{db: db}
}

func (r *PostgresApplicationRepository) CreateApplication(ctx context.Context, application *models.Application) (*models.Application, error) {
	return scanApplication(r.db.QueryRow(ctx, `
		INSERT INTO applications (
			user_id,
			job_id,
			employer_id,
			status,
			source
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING `+applicationColumns,
		application.UserID,
		application.JobID,
		application.EmployerID,
		application.Status,
		application.Source,
	))
}

func (r *PostgresApplicationRepository) GetApplicationByID(ctx context.Context, id string) (*models.Application, error) {
	return scanApplication(r.db.QueryRow(ctx, `SELECT `+applicationColumns+` FROM applications WHERE id = $1`, id))
}

func (r *PostgresApplicationRepository) ListApplicationsByUser(ctx context.Context, userID string, limit, offset int) ([]models.Application, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+applicationColumns+`
		FROM applications
		WHERE user_id = $1
		ORDER BY applied_at DESC
		LIMIT $2 OFFSET $3`,
		userID,
		normalizeLimit(limit),
		normalizeOffset(offset),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanApplications(rows)
}

func (r *PostgresApplicationRepository) ListApplicationsByJob(ctx context.Context, jobID string, limit, offset int) ([]models.Application, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+applicationColumns+`
		FROM applications
		WHERE job_id = $1
		ORDER BY applied_at DESC
		LIMIT $2 OFFSET $3`,
		jobID,
		normalizeLimit(limit),
		normalizeOffset(offset),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanApplications(rows)
}

func (r *PostgresApplicationRepository) UpdateApplicationStatus(ctx context.Context, id string, status models.ApplicationStatus) (*models.Application, error) {
	return scanApplication(r.db.QueryRow(ctx, `
		UPDATE applications
		SET status = $2
		WHERE id = $1
		RETURNING `+applicationColumns,
		id,
		status,
	))
}

func scanApplications(rows pgx.Rows) ([]models.Application, error) {
	var applications []models.Application
	for rows.Next() {
		application, err := scanApplication(rows)
		if err != nil {
			return nil, err
		}
		applications = append(applications, *application)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return applications, nil
}

func scanApplication(row interface{ Scan(dest ...any) error }) (*models.Application, error) {
	var application models.Application
	err := row.Scan(
		&application.ID,
		&application.UserID,
		&application.JobID,
		&application.EmployerID,
		&application.Status,
		&application.Source,
		&application.AppliedAt,
		&application.CreatedAt,
		&application.UpdatedAt,
	)
	if err != nil {
		return nil, mapNotFound(err)
	}

	return &application, nil
}
