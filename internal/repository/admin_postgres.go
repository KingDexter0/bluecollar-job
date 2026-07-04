package repository

import (
	"context"

	"bluecollarjob/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresAdminRepository struct {
	db *pgxpool.Pool
}

func NewPostgresAdminRepository(db *pgxpool.Pool) *PostgresAdminRepository {
	return &PostgresAdminRepository{db: db}
}

func (r *PostgresAdminRepository) GetSummary(ctx context.Context) (*models.AdminSummary, error) {
	summary := &models.AdminSummary{
		ApplicationsByStatus:      make(map[models.ApplicationStatus]int64),
		WorkersByVerificationTier: make(map[models.VerificationTier]int64),
		JobsByActiveStatus:        make(map[string]int64),
		ReferralsByPayoutStatus:   make(map[string]int64),
	}

	if err := r.db.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM users),
			(SELECT COUNT(*) FROM employers),
			(SELECT COUNT(*) FROM jobs),
			(SELECT COUNT(*) FROM applications),
			(SELECT COUNT(*) FROM referrals),
			(SELECT COUNT(*) FROM notification_events),
			(SELECT COUNT(*) FROM notification_events WHERE status = 'Pending'),
			(SELECT COUNT(*) FROM notification_events WHERE status = 'Failed'),
			(SELECT COUNT(*) FROM referral_transactions WHERE status = 'Pending'),
			(SELECT COUNT(*) FROM referral_transactions WHERE status = 'Paid'),
			(SELECT COUNT(*) FROM referral_transactions WHERE status = 'Failed'),
			(SELECT COUNT(*) FROM applications WHERE status = 'Interview_Scheduled')
	`).Scan(
		&summary.TotalWorkers,
		&summary.TotalEmployers,
		&summary.TotalJobs,
		&summary.TotalApplications,
		&summary.TotalReferrals,
		&summary.TotalNotificationEvents,
		&summary.PendingNotifications,
		&summary.FailedNotifications,
		&summary.CashbackPending,
		&summary.CashbackPaid,
		&summary.CashbackFailed,
		&summary.InterviewsScheduled,
	); err != nil {
		return nil, err
	}

	if err := r.loadApplicationStatusCounts(ctx, summary); err != nil {
		return nil, err
	}
	if err := r.loadWorkerTierCounts(ctx, summary); err != nil {
		return nil, err
	}
	if err := r.loadJobActiveCounts(ctx, summary); err != nil {
		return nil, err
	}
	if err := r.loadReferralPayoutCounts(ctx, summary); err != nil {
		return nil, err
	}

	return summary, nil
}

func (r *PostgresAdminRepository) loadApplicationStatusCounts(ctx context.Context, summary *models.AdminSummary) error {
	rows, err := r.db.Query(ctx, `SELECT status, COUNT(*) FROM applications GROUP BY status`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var status models.ApplicationStatus
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			return err
		}
		summary.ApplicationsByStatus[status] = count
	}
	return rows.Err()
}

func (r *PostgresAdminRepository) loadWorkerTierCounts(ctx context.Context, summary *models.AdminSummary) error {
	rows, err := r.db.Query(ctx, `SELECT verification_tier, COUNT(*) FROM users GROUP BY verification_tier`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var tier models.VerificationTier
		var count int64
		if err := rows.Scan(&tier, &count); err != nil {
			return err
		}
		summary.WorkersByVerificationTier[tier] = count
	}
	return rows.Err()
}

func (r *PostgresAdminRepository) loadJobActiveCounts(ctx context.Context, summary *models.AdminSummary) error {
	rows, err := r.db.Query(ctx, `SELECT is_active, COUNT(*) FROM jobs GROUP BY is_active`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var active bool
		var count int64
		if err := rows.Scan(&active, &count); err != nil {
			return err
		}
		if active {
			summary.JobsByActiveStatus["active"] = count
		} else {
			summary.JobsByActiveStatus["inactive"] = count
		}
	}
	return rows.Err()
}

func (r *PostgresAdminRepository) loadReferralPayoutCounts(ctx context.Context, summary *models.AdminSummary) error {
	rows, err := r.db.Query(ctx, `SELECT status, COUNT(*) FROM referral_transactions GROUP BY status`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			return err
		}
		summary.ReferralsByPayoutStatus[status] = count
	}
	return rows.Err()
}
