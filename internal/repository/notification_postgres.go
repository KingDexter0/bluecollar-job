package repository

import (
	"context"
	"database/sql"

	"bluecollarjob/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

const notificationEventColumns = `
	id,
	user_id,
	employer_id,
	application_id,
	channel,
	event_type,
	recipient,
	payload,
	status,
	attempts,
	scheduled_at,
	processed_at,
	last_error,
	created_at,
	updated_at`

type PostgresNotificationRepository struct {
	db queryer
}

func NewPostgresNotificationRepository(db *pgxpool.Pool) *PostgresNotificationRepository {
	return &PostgresNotificationRepository{db: db}
}

func (r *PostgresNotificationRepository) CreateNotificationEvent(ctx context.Context, event *models.NotificationEvent) (*models.NotificationEvent, error) {
	channel := event.Channel
	if channel == "" {
		channel = "whatsapp"
	}
	status := event.Status
	if status == "" {
		status = models.NotificationStatusPending
	}
	payload := event.Payload
	if len(payload) == 0 {
		payload = []byte("{}")
	}
	var scheduledAt any
	if !event.ScheduledAt.IsZero() {
		scheduledAt = event.ScheduledAt
	}

	return scanNotificationEvent(r.db.QueryRow(ctx, `
		INSERT INTO notification_events (
			user_id,
			employer_id,
			application_id,
			channel,
			event_type,
			recipient,
			payload,
			status,
			scheduled_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, COALESCE($9, NOW()))
		RETURNING `+notificationEventColumns,
		nullableString(event.UserID),
		nullableString(event.EmployerID),
		nullableString(event.ApplicationID),
		channel,
		event.EventType,
		event.Recipient,
		payload,
		status,
		scheduledAt,
	))
}

func scanNotificationEvent(row interface{ Scan(dest ...any) error }) (*models.NotificationEvent, error) {
	var event models.NotificationEvent
	var userID, employerID, applicationID, lastError sql.NullString
	var processedAt sql.NullTime

	err := row.Scan(
		&event.ID,
		&userID,
		&employerID,
		&applicationID,
		&event.Channel,
		&event.EventType,
		&event.Recipient,
		&event.Payload,
		&event.Status,
		&event.Attempts,
		&event.ScheduledAt,
		&processedAt,
		&lastError,
		&event.CreatedAt,
		&event.UpdatedAt,
	)
	if err != nil {
		return nil, mapNotFound(err)
	}

	event.UserID = stringPtr(userID)
	event.EmployerID = stringPtr(employerID)
	event.ApplicationID = stringPtr(applicationID)
	event.ProcessedAt = timePtr(processedAt)
	event.LastError = stringPtr(lastError)

	return &event, nil
}
