package repository

import (
	"context"
	"time"

	"bluecollarjob/internal/models"
)

type UserRepository interface {
	CreateUser(ctx context.Context, user *models.User) (*models.User, error)
	GetUserByID(ctx context.Context, id string) (*models.User, error)
	GetUserByPhone(ctx context.Context, phoneNumber string) (*models.User, error)
	UpdateUserVerificationTier(ctx context.Context, id string, tier models.VerificationTier) (*models.User, error)
	GetUserByReferralCode(ctx context.Context, referralCode string) (*models.User, error)
}

type IdentityVerificationRepository interface {
	CreateVerificationRecord(ctx context.Context, verification *models.WorkerIdentityVerification) (*models.WorkerIdentityVerification, error)
	GetLatestVerificationByUserID(ctx context.Context, userID string) (*models.WorkerIdentityVerification, error)
	MarkOTPVerificationPending(ctx context.Context, id string, aadhaarLast4, aadhaarMasked, aadhaarHash, aadhaarReferenceKey string, consentGivenAt time.Time) (*models.WorkerIdentityVerification, error)
	MarkVerified(ctx context.Context, id string) (*models.WorkerIdentityVerification, error)
	MarkDocumentUploaded(ctx context.Context, id string, documentRef string) (*models.WorkerIdentityVerification, error)
	MarkSkipped(ctx context.Context, id string, reason string) (*models.WorkerIdentityVerification, error)
	MarkFailed(ctx context.Context, id string, reason string) (*models.WorkerIdentityVerification, error)
}

type EmployerRepository interface {
	CreateEmployer(ctx context.Context, employer *models.Employer) (*models.Employer, error)
	GetEmployerByID(ctx context.Context, id string) (*models.Employer, error)
	GetEmployerByEmail(ctx context.Context, email string) (*models.Employer, error)
}

type JobRepository interface {
	CreateJob(ctx context.Context, job *models.Job) (*models.Job, error)
	GetJobByID(ctx context.Context, id string) (*models.Job, error)
	ListActiveJobs(ctx context.Context, limit, offset int) ([]models.Job, error)
	ListJobsByEmployer(ctx context.Context, employerID string, limit, offset int) ([]models.Job, error)
	UpdateJobStatus(ctx context.Context, id string, isActive bool) (*models.Job, error)
}

type ApplicationRepository interface {
	CreateApplication(ctx context.Context, application *models.Application) (*models.Application, error)
	GetApplicationByID(ctx context.Context, id string) (*models.Application, error)
	ListApplicationsByUser(ctx context.Context, userID string, limit, offset int) ([]models.Application, error)
	ListApplicationsByJob(ctx context.Context, jobID string, limit, offset int) ([]models.Application, error)
	UpdateApplicationStatus(ctx context.Context, id string, status models.ApplicationStatus) (*models.Application, error)
}
