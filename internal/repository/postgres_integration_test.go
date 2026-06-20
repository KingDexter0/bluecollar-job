package repository

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"bluecollarjob/internal/database"
	"bluecollarjob/internal/models"
)

func TestPostgresRepositories(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db, err := database.NewPostgresPool(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	defer db.Close()

	repos := NewPostgresRepositories(db)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	targetRole := "Welder"
	preferredZone := "Pune"
	phone := "+9199" + suffix[len(suffix)-8:]
	referralCode := "TEST" + suffix[len(suffix)-8:]

	user, err := repos.Users.CreateUser(ctx, &models.User{
		PhoneNumber:        phone,
		FullName:           "Repository Test Worker",
		LanguagePreference: "en",
		TargetRole:         &targetRole,
		PreferredZone:      &preferredZone,
		VerificationTier:   models.VerificationTierHigh,
		ReferralCode:       referralCode,
		IsActive:           true,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	byPhone, err := repos.Users.GetUserByPhone(ctx, phone)
	if err != nil {
		t.Fatalf("get user by phone: %v", err)
	}
	if byPhone.ID != user.ID {
		t.Fatalf("expected user id %s, got %s", user.ID, byPhone.ID)
	}

	byReferralCode, err := repos.Users.GetUserByReferralCode(ctx, referralCode)
	if err != nil {
		t.Fatalf("get user by referral code: %v", err)
	}
	if byReferralCode.ID != user.ID {
		t.Fatalf("expected referral user id %s, got %s", user.ID, byReferralCode.ID)
	}

	updatedUser, err := repos.Users.UpdateUserVerificationTier(ctx, user.ID, models.VerificationTierLow)
	if err != nil {
		t.Fatalf("update user verification tier: %v", err)
	}
	if updatedUser.VerificationTier != models.VerificationTierLow {
		t.Fatalf("expected low verification tier, got %s", updatedUser.VerificationTier)
	}

	verification, err := repos.IdentityVerifications.CreateVerificationRecord(ctx, &models.WorkerIdentityVerification{
		UserID:       user.ID,
		Method:       models.IdentityVerificationMethodSkipped,
		Status:       models.IdentityVerificationStatusSkipped,
		ConsentGiven: false,
	})
	if err != nil {
		t.Fatalf("create verification: %v", err)
	}

	consentAt := time.Now().UTC()
	verification, err = repos.IdentityVerifications.MarkOTPVerificationPending(
		ctx,
		verification.ID,
		"4321",
		"XXXX-XXXX-4321",
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"mock-test-reference",
		consentAt,
	)
	if err != nil {
		t.Fatalf("mark otp pending: %v", err)
	}
	if verification.Status != models.IdentityVerificationStatusOTPSent {
		t.Fatalf("expected OTP_Sent status, got %s", verification.Status)
	}

	verification, err = repos.IdentityVerifications.MarkVerified(ctx, verification.ID)
	if err != nil {
		t.Fatalf("mark verified: %v", err)
	}
	if verification.VerifiedAt == nil {
		t.Fatal("expected verified timestamp")
	}

	latestVerification, err := repos.IdentityVerifications.GetLatestVerificationByUserID(ctx, user.ID)
	if err != nil {
		t.Fatalf("get latest verification: %v", err)
	}
	if latestVerification.ID != verification.ID {
		t.Fatalf("expected latest verification %s, got %s", verification.ID, latestVerification.ID)
	}

	employerEmail := "repo-test-" + suffix + "@example.com"
	employer, err := repos.Employers.CreateEmployer(ctx, &models.Employer{
		CompanyName: "Repository Test Employer",
		ContactName: "Test Recruiter",
		Email:       employerEmail,
		IsVerified:  true,
	})
	if err != nil {
		t.Fatalf("create employer: %v", err)
	}

	byEmail, err := repos.Employers.GetEmployerByEmail(ctx, employerEmail)
	if err != nil {
		t.Fatalf("get employer by email: %v", err)
	}
	if byEmail.ID != employer.ID {
		t.Fatalf("expected employer id %s, got %s", employer.ID, byEmail.ID)
	}

	wageMin := 2000000
	wageMax := 2600000
	publishedAt := time.Now().UTC()
	job, err := repos.Jobs.CreateJob(ctx, &models.Job{
		EmployerID:               employer.ID,
		Title:                    "Repository Test Job",
		Description:              "Created by repository integration test.",
		SkillCategory:            "Welding",
		LocationCity:             "Pune",
		LocationState:            "Maharashtra",
		WageMinPaise:             &wageMin,
		WageMaxPaise:             &wageMax,
		RequiredVerificationTier: models.VerificationTierLow,
		Openings:                 1,
		IsActive:                 true,
		PublishedAt:              &publishedAt,
	})
	if err != nil {
		t.Fatalf("create job: %v", err)
	}

	activeJobs, err := repos.Jobs.ListActiveJobs(ctx, 10, 0)
	if err != nil {
		t.Fatalf("list active jobs: %v", err)
	}
	if len(activeJobs) == 0 {
		t.Fatal("expected at least one active job")
	}

	employerJobs, err := repos.Jobs.ListJobsByEmployer(ctx, employer.ID, 10, 0)
	if err != nil {
		t.Fatalf("list jobs by employer: %v", err)
	}
	if len(employerJobs) != 1 {
		t.Fatalf("expected one employer job, got %d", len(employerJobs))
	}

	job, err = repos.Jobs.UpdateJobStatus(ctx, job.ID, false)
	if err != nil {
		t.Fatalf("update job status: %v", err)
	}
	if job.IsActive {
		t.Fatal("expected inactive job")
	}

	application, err := repos.Applications.CreateApplication(ctx, &models.Application{
		UserID:     user.ID,
		JobID:      job.ID,
		EmployerID: employer.ID,
		Status:     models.ApplicationStatusApplied,
		Source:     "test",
	})
	if err != nil {
		t.Fatalf("create application: %v", err)
	}

	applicationsByUser, err := repos.Applications.ListApplicationsByUser(ctx, user.ID, 10, 0)
	if err != nil {
		t.Fatalf("list applications by user: %v", err)
	}
	if len(applicationsByUser) != 1 {
		t.Fatalf("expected one user application, got %d", len(applicationsByUser))
	}

	applicationsByJob, err := repos.Applications.ListApplicationsByJob(ctx, job.ID, 10, 0)
	if err != nil {
		t.Fatalf("list applications by job: %v", err)
	}
	if len(applicationsByJob) != 1 {
		t.Fatalf("expected one job application, got %d", len(applicationsByJob))
	}

	application, err = repos.Applications.UpdateApplicationStatus(ctx, application.ID, models.ApplicationStatusShortlisted)
	if err != nil {
		t.Fatalf("update application status: %v", err)
	}
	if application.Status != models.ApplicationStatusShortlisted {
		t.Fatalf("expected shortlisted application, got %s", application.Status)
	}
}
