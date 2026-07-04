package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"bluecollarjob/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const applicationATSColumns = `
	a.id,
	a.user_id,
	a.job_id,
	a.employer_id,
	a.status,
	a.source,
	a.applied_at,
	a.created_at,
	a.updated_at,
	u.full_name,
	u.phone_number,
	u.verification_tier,
	u.target_role,
	u.preferred_zone,
	j.title,
	j.role`

const interviewSlotColumns = `
	id,
	application_id,
	starts_at,
	ends_at,
	timezone,
	factory_location,
	google_maps_url,
	status,
	locked_until,
	confirmed_at,
	created_at,
	updated_at`

const interviewSlotAliasColumns = `
	s.id,
	s.application_id,
	s.starts_at,
	s.ends_at,
	s.timezone,
	s.factory_location,
	s.google_maps_url,
	s.status,
	s.locked_until,
	s.confirmed_at,
	s.created_at,
	s.updated_at`

type PostgresATSRepository struct {
	db *pgxpool.Pool
}

func NewPostgresATSRepository(db *pgxpool.Pool) *PostgresATSRepository {
	return &PostgresATSRepository{db: db}
}

func (r *PostgresATSRepository) ListEmployerApplications(ctx context.Context, employerID string, filters EmployerApplicationFilters) ([]models.ApplicationATS, error) {
	args := []any{employerID}
	conditions := []string{"a.employer_id = $1"}

	if filters.JobID != nil && *filters.JobID != "" {
		args = append(args, *filters.JobID)
		conditions = append(conditions, fmt.Sprintf("a.job_id = $%d", len(args)))
	}
	if filters.Status != nil && *filters.Status != "" {
		args = append(args, *filters.Status)
		conditions = append(conditions, fmt.Sprintf("a.status = $%d", len(args)))
	}
	if filters.VerificationTier != nil && *filters.VerificationTier != "" {
		args = append(args, *filters.VerificationTier)
		conditions = append(conditions, fmt.Sprintf("u.verification_tier = $%d", len(args)))
	}
	if filters.TargetRole != nil && *filters.TargetRole != "" {
		args = append(args, "%"+strings.ToLower(*filters.TargetRole)+"%")
		conditions = append(conditions, fmt.Sprintf("LOWER(COALESCE(u.target_role, '')) LIKE $%d", len(args)))
	}
	if filters.PreferredZone != nil && *filters.PreferredZone != "" {
		args = append(args, "%"+strings.ToLower(*filters.PreferredZone)+"%")
		conditions = append(conditions, fmt.Sprintf("LOWER(COALESCE(u.preferred_zone, '')) LIKE $%d", len(args)))
	}

	args = append(args, normalizeLimit(filters.Limit), normalizeOffset(filters.Offset))
	query := `
		SELECT ` + applicationATSColumns + `
		FROM applications a
		INNER JOIN users u ON u.id = a.user_id
		INNER JOIN jobs j ON j.id = a.job_id
		WHERE ` + strings.Join(conditions, " AND ") + `
		ORDER BY a.applied_at DESC
		LIMIT $` + fmt.Sprint(len(args)-1) + ` OFFSET $` + fmt.Sprint(len(args))

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var applications []models.ApplicationATS
	for rows.Next() {
		application, err := scanApplicationATS(rows)
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

func (r *PostgresATSRepository) GetEmployerApplicationByID(ctx context.Context, employerID, applicationID string) (*models.ApplicationATS, error) {
	return getEmployerApplicationByID(ctx, r.db, employerID, applicationID)
}

func (r *PostgresATSRepository) UpdateEmployerApplicationStatus(ctx context.Context, employerID, applicationID string, status models.ApplicationStatus) (*models.ApplicationATS, error) {
	command, err := r.db.Exec(ctx, `
		UPDATE applications
		SET status = $3
		WHERE employer_id = $1 AND id = $2`,
		employerID,
		applicationID,
		status,
	)
	if err != nil {
		return nil, err
	}
	if command.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	return getEmployerApplicationByID(ctx, r.db, employerID, applicationID)
}

func (r *PostgresATSRepository) CreateDirectInterview(ctx context.Context, employerID, applicationID string, slot InterviewSlotInput) (*models.InterviewSlot, *models.ApplicationATS, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, nil, err
	}
	defer rollbackUnlessCommitted(ctx, tx)

	if err := lockEmployerApplication(ctx, tx, employerID, applicationID); err != nil {
		return nil, nil, err
	}

	createdSlot, err := insertInterviewSlot(ctx, tx, applicationID, slot, models.InterviewSlotStatusConfirmed)
	if err != nil {
		return nil, nil, err
	}

	if _, err := tx.Exec(ctx, `
		UPDATE applications
		SET status = $3
		WHERE employer_id = $1 AND id = $2`,
		employerID,
		applicationID,
		models.ApplicationStatusInterviewScheduled,
	); err != nil {
		return nil, nil, err
	}

	application, err := getEmployerApplicationByID(ctx, tx, employerID, applicationID)
	if err != nil {
		return nil, nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, nil, err
	}
	return createdSlot, application, nil
}

func (r *PostgresATSRepository) CreateInterviewSlots(ctx context.Context, employerID, applicationID string, slots []InterviewSlotInput) ([]models.InterviewSlot, *models.ApplicationATS, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, nil, err
	}
	defer rollbackUnlessCommitted(ctx, tx)

	if err := lockEmployerApplication(ctx, tx, employerID, applicationID); err != nil {
		return nil, nil, err
	}

	createdSlots := make([]models.InterviewSlot, 0, len(slots))
	for _, slot := range slots {
		createdSlot, err := insertInterviewSlot(ctx, tx, applicationID, slot, models.InterviewSlotStatusAvailable)
		if err != nil {
			return nil, nil, err
		}
		createdSlots = append(createdSlots, *createdSlot)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE applications
		SET status = $3
		WHERE employer_id = $1 AND id = $2`,
		employerID,
		applicationID,
		models.ApplicationStatusSlotSelectionPending,
	); err != nil {
		return nil, nil, err
	}

	application, err := getEmployerApplicationByID(ctx, tx, employerID, applicationID)
	if err != nil {
		return nil, nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, nil, err
	}
	return createdSlots, application, nil
}

func (r *PostgresATSRepository) SelectInterviewSlot(ctx context.Context, applicationID, slotID string) (*models.InterviewSlot, *models.Application, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, nil, err
	}
	defer rollbackUnlessCommitted(ctx, tx)

	slot, err := scanInterviewSlot(tx.QueryRow(ctx, `
		SELECT `+interviewSlotAliasColumns+`
		FROM interview_slots s
		WHERE s.id = $1 AND s.application_id = $2
		FOR UPDATE`,
		slotID,
		applicationID,
	))
	if err != nil {
		return nil, nil, err
	}
	if slot.Status != models.InterviewSlotStatusAvailable {
		return nil, nil, ErrConflict
	}

	confirmedSlot, err := scanInterviewSlot(tx.QueryRow(ctx, `
		UPDATE interview_slots
		SET status = $3, confirmed_at = NOW()
		WHERE id = $1 AND application_id = $2 AND status = $4
		RETURNING `+interviewSlotColumns,
		slotID,
		applicationID,
		models.InterviewSlotStatusConfirmed,
		models.InterviewSlotStatusAvailable,
	))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, nil, ErrConflict
		}
		return nil, nil, err
	}

	if _, err := tx.Exec(ctx, `
		UPDATE interview_slots
		SET status = $3
		WHERE application_id = $1
			AND id <> $2
			AND status IN ($4, $5)`,
		applicationID,
		slotID,
		models.InterviewSlotStatusCancelled,
		models.InterviewSlotStatusAvailable,
		models.InterviewSlotStatusLocked,
	); err != nil {
		return nil, nil, err
	}

	application, err := scanApplication(tx.QueryRow(ctx, `
		UPDATE applications
		SET status = $2
		WHERE id = $1
		RETURNING `+applicationColumns,
		applicationID,
		models.ApplicationStatusInterviewScheduled,
	))
	if err != nil {
		return nil, nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, err
	}
	return confirmedSlot, application, nil
}

func getEmployerApplicationByID(ctx context.Context, db queryer, employerID, applicationID string) (*models.ApplicationATS, error) {
	return scanApplicationATS(db.QueryRow(ctx, `
		SELECT `+applicationATSColumns+`
		FROM applications a
		INNER JOIN users u ON u.id = a.user_id
		INNER JOIN jobs j ON j.id = a.job_id
		WHERE a.employer_id = $1 AND a.id = $2`,
		employerID,
		applicationID,
	))
}

func lockEmployerApplication(ctx context.Context, tx pgx.Tx, employerID, applicationID string) error {
	var id string
	err := tx.QueryRow(ctx, `
		SELECT id
		FROM applications
		WHERE employer_id = $1 AND id = $2
		FOR UPDATE`,
		employerID,
		applicationID,
	).Scan(&id)
	return mapNotFound(err)
}

func insertInterviewSlot(ctx context.Context, db queryer, applicationID string, slot InterviewSlotInput, status models.InterviewSlotStatus) (*models.InterviewSlot, error) {
	timezone := strings.TrimSpace(slot.Timezone)
	if timezone == "" {
		timezone = "Asia/Kolkata"
	}
	return scanInterviewSlot(db.QueryRow(ctx, `
		INSERT INTO interview_slots (
			application_id,
			starts_at,
			ends_at,
			timezone,
			factory_location,
			google_maps_url,
			status,
			confirmed_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, CASE WHEN $7::interview_slot_status_enum = 'Confirmed' THEN NOW() ELSE NULL END)
		RETURNING `+interviewSlotColumns,
		applicationID,
		slot.StartsAt,
		slot.EndsAt,
		timezone,
		nullableString(slot.FactoryLocation),
		nullableString(slot.GoogleMapsURL),
		status,
	))
}

func scanApplicationATS(row interface{ Scan(dest ...any) error }) (*models.ApplicationATS, error) {
	var application models.ApplicationATS
	var targetRole, preferredZone sql.NullString
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
		&application.WorkerFullName,
		&application.WorkerPhoneNumber,
		&application.WorkerVerificationTier,
		&targetRole,
		&preferredZone,
		&application.JobTitle,
		&application.JobRole,
	)
	if err != nil {
		return nil, mapNotFound(err)
	}
	application.WorkerTargetRole = stringPtr(targetRole)
	application.WorkerPreferredZone = stringPtr(preferredZone)
	return &application, nil
}

func scanInterviewSlot(row interface{ Scan(dest ...any) error }) (*models.InterviewSlot, error) {
	var slot models.InterviewSlot
	var factoryLocation, googleMapsURL sql.NullString
	var lockedUntil, confirmedAt sql.NullTime
	err := row.Scan(
		&slot.ID,
		&slot.ApplicationID,
		&slot.StartsAt,
		&slot.EndsAt,
		&slot.Timezone,
		&factoryLocation,
		&googleMapsURL,
		&slot.Status,
		&lockedUntil,
		&confirmedAt,
		&slot.CreatedAt,
		&slot.UpdatedAt,
	)
	if err != nil {
		return nil, mapNotFound(err)
	}
	slot.FactoryLocation = stringPtr(factoryLocation)
	slot.GoogleMapsURL = stringPtr(googleMapsURL)
	slot.LockedUntil = timePtr(lockedUntil)
	slot.ConfirmedAt = timePtr(confirmedAt)
	return &slot, nil
}

func rollbackUnlessCommitted(ctx context.Context, tx pgx.Tx) {
	_ = tx.Rollback(ctx)
}
