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
	UpdateUserProfile(ctx context.Context, user *models.User) (*models.User, error)
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
	UpdateEmployerProfile(ctx context.Context, employer *models.Employer) (*models.Employer, error)
	GetActiveSubscriptionTier(ctx context.Context, employerID string) (models.SubscriptionTier, error)
}

type JobRepository interface {
	CreateJob(ctx context.Context, job *models.Job) (*models.Job, error)
	GetJobByID(ctx context.Context, id string) (*models.Job, error)
	GetJobByIDAndEmployer(ctx context.Context, id, employerID string) (*models.Job, error)
	ListActiveJobs(ctx context.Context, limit, offset int) ([]models.Job, error)
	ListJobsByEmployer(ctx context.Context, employerID string, limit, offset int) ([]models.Job, error)
	CountActiveJobsByEmployer(ctx context.Context, employerID string) (int, error)
	UpdateJob(ctx context.Context, job *models.Job) (*models.Job, error)
	UpdateJobStatus(ctx context.Context, id string, isActive bool) (*models.Job, error)
	UpdateEmployerJobStatus(ctx context.Context, id, employerID string, isActive bool) (*models.Job, error)
}

type ApplicationRepository interface {
	CreateApplication(ctx context.Context, application *models.Application) (*models.Application, error)
	GetApplicationByID(ctx context.Context, id string) (*models.Application, error)
	ListApplicationsByUser(ctx context.Context, userID string, limit, offset int) ([]models.Application, error)
	ListApplicationsByJob(ctx context.Context, jobID string, limit, offset int) ([]models.Application, error)
	UpdateApplicationStatus(ctx context.Context, id string, status models.ApplicationStatus) (*models.Application, error)
}

type EmployerApplicationFilters struct {
	JobID            *string
	Status           *models.ApplicationStatus
	VerificationTier *models.VerificationTier
	TargetRole       *string
	PreferredZone    *string
	Limit            int
	Offset           int
}

type InterviewSlotInput struct {
	StartsAt        time.Time
	EndsAt          time.Time
	Timezone        string
	FactoryLocation *string
	GoogleMapsURL   *string
}

type ATSRepository interface {
	ListEmployerApplications(ctx context.Context, employerID string, filters EmployerApplicationFilters) ([]models.ApplicationATS, error)
	GetEmployerApplicationByID(ctx context.Context, employerID, applicationID string) (*models.ApplicationATS, error)
	UpdateEmployerApplicationStatus(ctx context.Context, employerID, applicationID string, status models.ApplicationStatus) (*models.ApplicationATS, error)
	CreateDirectInterview(ctx context.Context, employerID, applicationID string, slot InterviewSlotInput) (*models.InterviewSlot, *models.ApplicationATS, error)
	CreateInterviewSlots(ctx context.Context, employerID, applicationID string, slots []InterviewSlotInput) ([]models.InterviewSlot, *models.ApplicationATS, error)
	SelectInterviewSlot(ctx context.Context, applicationID, slotID string) (*models.InterviewSlot, *models.Application, error)
}

type NotificationRepository interface {
	CreateNotificationEvent(ctx context.Context, event *models.NotificationEvent) (*models.NotificationEvent, error)
	ClaimPendingNotificationEvents(ctx context.Context, limit int) ([]models.NotificationEvent, error)
	ListNotificationEvents(ctx context.Context, filters NotificationEventFilters) ([]models.NotificationEvent, error)
	MarkNotificationEventSent(ctx context.Context, id string) (*models.NotificationEvent, error)
	MarkNotificationEventFailed(ctx context.Context, id string, reason string) (*models.NotificationEvent, error)
}

type NotificationEventFilters struct {
	Status    *models.NotificationStatus
	EventType *string
	Limit     int
	Offset    int
}

type ReferralRepository interface {
	CreateReferral(ctx context.Context, referral *models.Referral) (*models.Referral, error)
	GetReferralByReferredUserID(ctx context.Context, referredUserID string) (*models.Referral, error)
	ListReferralsByReferrer(ctx context.Context, referrerUserID string, limit, offset int) ([]models.Referral, error)
	ListReferralTransactionsByUser(ctx context.Context, userID string, limit, offset int) ([]models.ReferralTransaction, error)
	ListReferralTransactions(ctx context.Context, filters ReferralTransactionFilters) ([]models.ReferralTransaction, error)
	MarkReferralConverted(ctx context.Context, id string) (*models.Referral, error)
	CreateReferralTransaction(ctx context.Context, transaction *models.ReferralTransaction) (*models.ReferralTransaction, error)
	ClaimPendingReferralTransactions(ctx context.Context, limit int) ([]models.ReferralTransaction, error)
	MarkReferralTransactionPaid(ctx context.Context, id, externalReference string) (*models.ReferralTransaction, error)
	MarkReferralTransactionFailed(ctx context.Context, id, reason string) (*models.ReferralTransaction, error)
}

type ReferralTransactionFilters struct {
	Status *string
	Limit  int
	Offset int
}

type AdminRepository interface {
	GetSummary(ctx context.Context) (*models.AdminSummary, error)
}
