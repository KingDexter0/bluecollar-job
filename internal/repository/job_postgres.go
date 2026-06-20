package repository

import (
	"context"
	"database/sql"

	"bluecollarjob/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const jobColumns = `
	id,
	employer_id,
	title,
	description,
	skill_category,
	location_city,
	location_state,
	wage_min_paise,
	wage_max_paise,
	required_verification_tier,
	openings,
	is_active,
	published_at,
	expires_at,
	created_at,
	updated_at`

type PostgresJobRepository struct {
	db queryer
}

func NewPostgresJobRepository(db *pgxpool.Pool) *PostgresJobRepository {
	return &PostgresJobRepository{db: db}
}

func (r *PostgresJobRepository) CreateJob(ctx context.Context, job *models.Job) (*models.Job, error) {
	return scanJob(r.db.QueryRow(ctx, `
		INSERT INTO jobs (
			employer_id,
			title,
			description,
			skill_category,
			location_city,
			location_state,
			wage_min_paise,
			wage_max_paise,
			required_verification_tier,
			openings,
			is_active,
			published_at,
			expires_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING `+jobColumns,
		job.EmployerID,
		job.Title,
		job.Description,
		job.SkillCategory,
		job.LocationCity,
		job.LocationState,
		nullableInt(job.WageMinPaise),
		nullableInt(job.WageMaxPaise),
		job.RequiredVerificationTier,
		job.Openings,
		job.IsActive,
		nullableTime(job.PublishedAt),
		nullableTime(job.ExpiresAt),
	))
}

func (r *PostgresJobRepository) GetJobByID(ctx context.Context, id string) (*models.Job, error) {
	return scanJob(r.db.QueryRow(ctx, `SELECT `+jobColumns+` FROM jobs WHERE id = $1`, id))
}

func (r *PostgresJobRepository) ListActiveJobs(ctx context.Context, limit, offset int) ([]models.Job, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+jobColumns+`
		FROM jobs
		WHERE is_active = TRUE
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`,
		normalizeLimit(limit),
		normalizeOffset(offset),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanJobs(rows)
}

func (r *PostgresJobRepository) ListJobsByEmployer(ctx context.Context, employerID string, limit, offset int) ([]models.Job, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+jobColumns+`
		FROM jobs
		WHERE employer_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`,
		employerID,
		normalizeLimit(limit),
		normalizeOffset(offset),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanJobs(rows)
}

func (r *PostgresJobRepository) UpdateJobStatus(ctx context.Context, id string, isActive bool) (*models.Job, error) {
	return scanJob(r.db.QueryRow(ctx, `
		UPDATE jobs
		SET is_active = $2
		WHERE id = $1
		RETURNING `+jobColumns,
		id,
		isActive,
	))
}

func scanJobs(rows pgx.Rows) ([]models.Job, error) {
	var jobs []models.Job
	for rows.Next() {
		job, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, *job)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return jobs, nil
}

func scanJob(row interface{ Scan(dest ...any) error }) (*models.Job, error) {
	var job models.Job
	var wageMinPaise sql.NullInt32
	var wageMaxPaise sql.NullInt32
	var publishedAt sql.NullTime
	var expiresAt sql.NullTime

	err := row.Scan(
		&job.ID,
		&job.EmployerID,
		&job.Title,
		&job.Description,
		&job.SkillCategory,
		&job.LocationCity,
		&job.LocationState,
		&wageMinPaise,
		&wageMaxPaise,
		&job.RequiredVerificationTier,
		&job.Openings,
		&job.IsActive,
		&publishedAt,
		&expiresAt,
		&job.CreatedAt,
		&job.UpdatedAt,
	)
	if err != nil {
		return nil, mapNotFound(err)
	}

	job.WageMinPaise = intPtr(wageMinPaise)
	job.WageMaxPaise = intPtr(wageMaxPaise)
	job.PublishedAt = timePtr(publishedAt)
	job.ExpiresAt = timePtr(expiresAt)

	return &job, nil
}
