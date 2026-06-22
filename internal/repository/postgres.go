package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound = errors.New("record not found")
	ErrConflict = errors.New("record conflict")
)

type PostgresRepositories struct {
	Users                 UserRepository
	IdentityVerifications IdentityVerificationRepository
	Employers             EmployerRepository
	Jobs                  JobRepository
	Applications          ApplicationRepository
	ATS                   ATSRepository
	Notifications         NotificationRepository
}

func NewPostgresRepositories(db *pgxpool.Pool) *PostgresRepositories {
	return &PostgresRepositories{
		Users:                 NewPostgresUserRepository(db),
		IdentityVerifications: NewPostgresIdentityVerificationRepository(db),
		Employers:             NewPostgresEmployerRepository(db),
		Jobs:                  NewPostgresJobRepository(db),
		Applications:          NewPostgresApplicationRepository(db),
		ATS:                   NewPostgresATSRepository(db),
		Notifications:         NewPostgresNotificationRepository(db),
	}
}

type queryer interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func mapNotFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func stringPtr(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}

func intPtr(value sql.NullInt32) *int {
	if !value.Valid {
		return nil
	}
	i := int(value.Int32)
	return &i
}

func timePtr(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}

func nullableString(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableInt(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return *value
}

func normalizeLimit(limit int) int {
	if limit <= 0 || limit > 100 {
		return 50
	}
	return limit
}

func normalizeOffset(offset int) int {
	if offset < 0 {
		return 0
	}
	return offset
}
