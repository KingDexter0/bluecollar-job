package models

import "time"

type VerificationTier string

const (
	VerificationTierLow    VerificationTier = "Low"
	VerificationTierMedium VerificationTier = "Medium"
	VerificationTierHigh   VerificationTier = "High"
)

type IdentityVerificationMethod string

const (
	IdentityVerificationMethodAadhaarOTP     IdentityVerificationMethod = "Aadhaar_OTP"
	IdentityVerificationMethodDocumentUpload IdentityVerificationMethod = "Document_Upload"
	IdentityVerificationMethodSkipped        IdentityVerificationMethod = "Skipped"
)

type IdentityVerificationStatus string

const (
	IdentityVerificationStatusPending          IdentityVerificationStatus = "Pending"
	IdentityVerificationStatusOTPSent          IdentityVerificationStatus = "OTP_Sent"
	IdentityVerificationStatusVerified         IdentityVerificationStatus = "Verified"
	IdentityVerificationStatusFailed           IdentityVerificationStatus = "Failed"
	IdentityVerificationStatusDocumentUploaded IdentityVerificationStatus = "Document_Uploaded"
	IdentityVerificationStatusSkipped          IdentityVerificationStatus = "Skipped"
)

type ApplicationStatus string

const (
	ApplicationStatusApplied              ApplicationStatus = "Applied"
	ApplicationStatusShortlisted          ApplicationStatus = "Shortlisted"
	ApplicationStatusSlotSelectionPending ApplicationStatus = "Slot_Selection_Pending"
	ApplicationStatusInterviewScheduled   ApplicationStatus = "Interview_Scheduled"
	ApplicationStatusSelected             ApplicationStatus = "Selected"
	ApplicationStatusRejected             ApplicationStatus = "Rejected"
)

type SubscriptionTier string

const (
	SubscriptionTierGrowth     SubscriptionTier = "Growth"
	SubscriptionTierEnterprise SubscriptionTier = "Enterprise"
)

type InterviewSlotStatus string

const (
	InterviewSlotStatusAvailable InterviewSlotStatus = "Available"
	InterviewSlotStatusLocked    InterviewSlotStatus = "Locked"
	InterviewSlotStatusConfirmed InterviewSlotStatus = "Confirmed"
	InterviewSlotStatusCancelled InterviewSlotStatus = "Cancelled"
)

type NotificationStatus string

const (
	NotificationStatusPending    NotificationStatus = "Pending"
	NotificationStatusProcessing NotificationStatus = "Processing"
	NotificationStatusSent       NotificationStatus = "Sent"
	NotificationStatusFailed     NotificationStatus = "Failed"
)

type User struct {
	ID                 string
	PhoneNumber        string
	FullName           string
	LanguagePreference string
	TargetRole         *string
	PreferredZone      *string
	VerificationTier   VerificationTier
	ReferralCode       string
	ReferredByCode     *string
	IsActive           bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type WorkerIdentityVerification struct {
	ID                  string
	UserID              string
	Method              IdentityVerificationMethod
	Status              IdentityVerificationStatus
	AadhaarLast4        *string
	AadhaarMasked       *string
	AadhaarHash         *string
	AadhaarReferenceKey *string
	DocumentRef         *string
	ConsentGiven        bool
	ConsentGivenAt      *time.Time
	VerifiedAt          *time.Time
	FailedReason        *string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type Employer struct {
	ID           string
	CompanyName  string
	ContactName  string
	Email        string
	PasswordHash string
	PhoneNumber  *string
	City         *string
	State        *string
	IsVerified   bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Job struct {
	ID                       string
	EmployerID               string
	Title                    string
	Role                     string
	Description              string
	SkillCategory            string
	LocationCity             string
	LocationState            string
	ShiftSchedule            string
	WageMinPaise             *int
	WageMaxPaise             *int
	RequiredVerificationTier VerificationTier
	Openings                 int
	IsActive                 bool
	PublishedAt              *time.Time
	ExpiresAt                *time.Time
	CreatedAt                time.Time
	UpdatedAt                time.Time
}

type Application struct {
	ID         string
	UserID     string
	JobID      string
	EmployerID string
	Status     ApplicationStatus
	Source     string
	AppliedAt  time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type InterviewSlot struct {
	ID            string
	ApplicationID string
	StartsAt      time.Time
	EndsAt        time.Time
	Timezone      string
	FactoryLocation *string
	GoogleMapsURL   *string
	Status          InterviewSlotStatus
	LockedUntil     *time.Time
	ConfirmedAt     *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type Referral struct {
	ID             string
	ReferrerUserID string
	ReferredUserID *string
	ReferralCode   string
	CreatedAt      time.Time
	ConvertedAt    *time.Time
}

type ReferralTransaction struct {
	ID                string
	ReferralID        string
	UserID            string
	AmountPaise       int
	Currency          string
	Status            string
	ExternalReference *string
	CreatedAt         time.Time
	PaidAt            *time.Time
}

type Subscription struct {
	ID           string
	EmployerID   string
	Tier         SubscriptionTier
	StartsAt     time.Time
	EndsAt       *time.Time
	JobPostLimit *int
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type NotificationEvent struct {
	ID            string
	UserID        *string
	EmployerID    *string
	ApplicationID *string
	Channel       string
	EventType     string
	Recipient     string
	Payload       []byte
	Status        NotificationStatus
	Attempts      int
	ScheduledAt   time.Time
	ProcessedAt   *time.Time
	LastError     *string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type ApplicationATS struct {
	Application
	WorkerFullName         string
	WorkerPhoneNumber      string
	WorkerVerificationTier VerificationTier
	WorkerTargetRole       *string
	WorkerPreferredZone    *string
	JobTitle               string
	JobRole                string
}
