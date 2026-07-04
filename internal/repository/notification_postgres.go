package repository

import (
	"context"
	"database/sql"
	"strings"

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

const notificationEventAliasColumns = `
	n.id,
	n.user_id,
	n.employer_id,
	n.application_id,
	n.channel,
	n.event_type,
	n.recipient,
	n.payload,
	n.status,
	n.attempts,
	n.scheduled_at,
	n.processed_at,
	n.last_error,
	n.created_at,
	n.updated_at`

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

func (r *PostgresNotificationRepository) ClaimPendingNotificationEvents(ctx context.Context, limit int) ([]models.NotificationEvent, error) {
	rows, err := r.db.Query(ctx, `
		WITH claimed AS (
			SELECT id
			FROM notification_events
			WHERE status = $1 AND scheduled_at <= NOW()
			ORDER BY scheduled_at ASC, created_at ASC
			LIMIT $2
			FOR UPDATE SKIP LOCKED
		)
		UPDATE notification_events n
		SET status = $3,
			attempts = attempts + 1,
			last_error = NULL
		FROM claimed
		WHERE n.id = claimed.id
		RETURNING `+notificationEventAliasColumns,
		models.NotificationStatusPending,
		normalizeLimit(limit),
		models.NotificationStatusProcessing,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []models.NotificationEvent
	for rows.Next() {
		event, err := scanNotificationEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, *event)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return events, nil
}

func (r *PostgresNotificationRepository) ListNotificationEvents(ctx context.Context, filters NotificationEventFilters) ([]models.NotificationEvent, error) {
	conditions := []string{"1=1"}
	args := []any{}
	if filters.Status != nil {
		args = append(args, *filters.Status)
		conditions = append(conditions, "n.status = $"+argPosition(len(args)))
	}
	if filters.EventType != nil && strings.TrimSpace(*filters.EventType) != "" {
		args = append(args, strings.TrimSpace(*filters.EventType))
		conditions = append(conditions, "n.event_type = $"+argPosition(len(args)))
	}

	args = append(args, normalizeLimit(filters.Limit), normalizeOffset(filters.Offset))
	limitArg := argPosition(len(args) - 1)
	offsetArg := argPosition(len(args))

	rows, err := r.db.Query(ctx, `
		SELECT `+notificationEventAliasColumns+`
		FROM notification_events n
		WHERE `+strings.Join(conditions, " AND ")+`
		ORDER BY n.created_at DESC, n.id DESC
		LIMIT $`+limitArg+` OFFSET $`+offsetArg,
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []models.NotificationEvent
	for rows.Next() {
		event, err := scanNotificationEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, *event)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return events, nil
}

func (r *PostgresNotificationRepository) MarkNotificationEventSent(ctx context.Context, id string) (*models.NotificationEvent, error) {
	return scanNotificationEvent(r.db.QueryRow(ctx, `
		UPDATE notification_events
		SET status = $2,
			processed_at = NOW(),
			last_error = NULL
		WHERE id = $1
		RETURNING `+notificationEventColumns,
		id,
		models.NotificationStatusSent,
	))
}

func (r *PostgresNotificationRepository) MarkNotificationEventFailed(ctx context.Context, id string, reason string) (*models.NotificationEvent, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "notification delivery failed"
	}
	return scanNotificationEvent(r.db.QueryRow(ctx, `
		UPDATE notification_events
		SET status = $2,
			processed_at = NOW(),
			last_error = $3
		WHERE id = $1
		RETURNING `+notificationEventColumns,
		id,
		models.NotificationStatusFailed,
		reason,
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
