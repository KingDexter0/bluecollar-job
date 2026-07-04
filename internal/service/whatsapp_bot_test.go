package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"bluecollarjob/internal/models"
	"bluecollarjob/internal/repository"
)

func TestWhatsAppBotNewUserGreetingFlow(t *testing.T) {
	ctx := context.Background()
	users := newBotUserRepository()
	bot := newTestWhatsAppBot(users, nil, newRecordingWhatsAppSender())

	reply, err := bot.HandleIncomingMessage(ctx, IncomingWhatsAppMessage{
		PhoneNumber: "+919876543220",
		Text:        "hi",
		MessageType: "text",
	})
	if err != nil {
		t.Fatalf("handle message: %v", err)
	}
	if reply.State != BotStateAwaitingName {
		t.Fatalf("expected awaiting name, got %s", reply.State)
	}
	if !strings.Contains(reply.Message, "full name") {
		t.Fatalf("expected name prompt, got %q", reply.Message)
	}
	if _, err := users.GetUserByPhone(ctx, "+919876543220"); err != nil {
		t.Fatalf("expected worker created: %v", err)
	}
}

func TestWhatsAppBotReturningUserMenuFlow(t *testing.T) {
	ctx := context.Background()
	users := newBotUserRepository()
	user := users.mustCreate("+919876543221", "Returning Worker")
	bot := newTestWhatsAppBot(users, nil, newRecordingWhatsAppSender())

	reply, err := bot.HandleIncomingMessage(ctx, IncomingWhatsAppMessage{PhoneNumber: user.PhoneNumber, Text: "menu"})
	if err != nil {
		t.Fatalf("handle message: %v", err)
	}
	if reply.State != BotStateReturningMenu {
		t.Fatalf("expected returning menu, got %s", reply.State)
	}
	if !strings.Contains(reply.Message, "Check Application Status") {
		t.Fatalf("expected menu, got %q", reply.Message)
	}
}

func TestWhatsAppBotLanguageSelection(t *testing.T) {
	ctx := context.Background()
	users := newBotUserRepository()
	user := users.mustCreate("+919876543222", "Worker")
	stateStore := NewRedisConversationStateService(newFakeRedisStore())
	sender := newRecordingWhatsAppSender()
	bot := NewWhatsAppBotService(users, newBotApplicationService(nil), newBotJobService(nil), newBotIdentityService(users), newBotReferralService(), stateStore, NewRedisStatusOTPService(newFakeRedisStore(), "pepper"), sender)

	if _, err := stateStore.SetState(ctx, user.PhoneNumber, BotStateAwaitingLanguage, mustRaw(botStateData{WorkerID: user.ID, FullName: "Worker", Language: "en"}), time.Hour); err != nil {
		t.Fatalf("seed state: %v", err)
	}
	reply, err := bot.HandleIncomingMessage(ctx, IncomingWhatsAppMessage{PhoneNumber: user.PhoneNumber, Text: "2"})
	if err != nil {
		t.Fatalf("handle language: %v", err)
	}
	if reply.State != BotStateAwaitingTargetRole {
		t.Fatalf("expected target role state, got %s", reply.State)
	}
	updated, _ := users.GetUserByPhone(ctx, user.PhoneNumber)
	if updated.LanguagePreference != "hi" {
		t.Fatalf("expected Hindi preference, got %s", updated.LanguagePreference)
	}
}

func TestWhatsAppBotProfileSetupStateTransitions(t *testing.T) {
	ctx := context.Background()
	users := newBotUserRepository()
	phone := "+919876543223"
	bot := newTestWhatsAppBot(users, nil, newRecordingWhatsAppSender())

	steps := []struct {
		text      string
		wantState string
	}{
		{text: "hello", wantState: BotStateAwaitingName},
		{text: "Ravi Kumar", wantState: BotStateAwaitingLanguage},
		{text: "1", wantState: BotStateAwaitingTargetRole},
		{text: "Machine Operator", wantState: BotStateAwaitingPreferredZone},
		{text: "Pune", wantState: BotStateAwaitingVerificationOption},
		{text: "C", wantState: BotStateAwaitingSkipConfirm},
		{text: "YES", wantState: BotStateReturningMenu},
	}
	for _, step := range steps {
		reply, err := bot.HandleIncomingMessage(ctx, IncomingWhatsAppMessage{PhoneNumber: phone, Text: step.text})
		if err != nil {
			t.Fatalf("step %q: %v", step.text, err)
		}
		if reply.State != step.wantState {
			t.Fatalf("step %q: expected %s, got %s", step.text, step.wantState, reply.State)
		}
	}

	user, err := users.GetUserByPhone(ctx, phone)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if user.FullName != "Ravi Kumar" || user.TargetRole == nil || *user.TargetRole != "Machine Operator" || user.PreferredZone == nil || *user.PreferredZone != "Pune" {
		t.Fatalf("profile not updated correctly: %#v", user)
	}
}

func TestWhatsAppBotApplicationStatusOTPGenerationAndVerification(t *testing.T) {
	ctx := context.Background()
	users := newBotUserRepository()
	user := users.mustCreate("+919876543224", "Status Worker")
	apps := newBotApplicationService([]models.Application{{
		ID:         "application-1",
		UserID:     user.ID,
		JobID:      "job-1",
		EmployerID: "employer-1",
		Status:     models.ApplicationStatusShortlisted,
		Source:     "test",
	}})
	bot := newTestWhatsAppBot(users, apps, newRecordingWhatsAppSender())

	menuReply, err := bot.HandleIncomingMessage(ctx, IncomingWhatsAppMessage{PhoneNumber: user.PhoneNumber, Text: "menu"})
	if err != nil {
		t.Fatalf("menu: %v", err)
	}
	if menuReply.State != BotStateReturningMenu {
		t.Fatalf("expected menu state, got %s", menuReply.State)
	}

	otpReply, err := bot.HandleIncomingMessage(ctx, IncomingWhatsAppMessage{PhoneNumber: user.PhoneNumber, Text: "1"})
	if err != nil {
		t.Fatalf("request otp: %v", err)
	}
	if otpReply.State != BotStateAwaitingStatusOTP || !strings.Contains(otpReply.Message, "Local dev OTP:") {
		t.Fatalf("expected OTP prompt with local OTP, got %#v", otpReply)
	}
	otp := strings.TrimSpace(strings.Split(otpReply.Message, "Local dev OTP:")[1])

	statusReply, err := bot.HandleIncomingMessage(ctx, IncomingWhatsAppMessage{PhoneNumber: user.PhoneNumber, Text: otp})
	if err != nil {
		t.Fatalf("verify otp: %v", err)
	}
	if statusReply.State != BotStateReturningMenu || !strings.Contains(statusReply.Message, string(models.ApplicationStatusShortlisted)) {
		t.Fatalf("expected application status reply, got %#v", statusReply)
	}
}

func TestWhatsAppBotAadhaarOTPVerificationFlow(t *testing.T) {
	ctx := context.Background()
	users := newBotUserRepository()
	phone := "+919876543225"
	identity := newBotIdentityService(users)
	bot := newTestWhatsAppBotWithDeps(users, nil, nil, identity, newRecordingWhatsAppSender())
	completeProfileSetup(t, ctx, bot, phone)

	steps := []struct {
		text      string
		wantState string
	}{
		{text: "A", wantState: BotStateAwaitingAadhaarNumber},
		{text: "123456789012", wantState: BotStateAwaitingAadhaarConsent},
		{text: "YES 123456789012", wantState: BotStateAwaitingAadhaarOTP},
		{text: "123456", wantState: BotStateReturningMenu},
	}
	for _, step := range steps {
		reply, err := bot.HandleIncomingMessage(ctx, IncomingWhatsAppMessage{PhoneNumber: phone, Text: step.text})
		if err != nil {
			t.Fatalf("step %q: %v", step.text, err)
		}
		if reply.State != step.wantState {
			t.Fatalf("step %q: expected %s, got %s", step.text, step.wantState, reply.State)
		}
	}
	user, _ := users.GetUserByPhone(ctx, phone)
	if user.VerificationTier != models.VerificationTierLow {
		t.Fatalf("expected low risk, got %s", user.VerificationTier)
	}
}

func TestWhatsAppBotDocumentUploadFlow(t *testing.T) {
	ctx := context.Background()
	users := newBotUserRepository()
	phone := "+919876543226"
	bot := newTestWhatsAppBot(users, nil, newRecordingWhatsAppSender())
	completeProfileSetup(t, ctx, bot, phone)

	reply, err := bot.HandleIncomingMessage(ctx, IncomingWhatsAppMessage{PhoneNumber: phone, Text: "B"})
	if err != nil || reply.State != BotStateAwaitingDocumentUpload {
		t.Fatalf("expected document state, reply=%#v err=%v", reply, err)
	}
	reply, err = bot.HandleIncomingMessage(ctx, IncomingWhatsAppMessage{PhoneNumber: phone, MessageType: "image"})
	if err != nil || reply.State != BotStateAwaitingDocumentUpload || !strings.Contains(reply.Message, "upload") {
		t.Fatalf("expected missing upload fallback, reply=%#v err=%v", reply, err)
	}
	ref := "mock-media/document-1"
	reply, err = bot.HandleIncomingMessage(ctx, IncomingWhatsAppMessage{PhoneNumber: phone, MessageType: "image", MediaRef: &ref})
	if err != nil || reply.State != BotStateReturningMenu {
		t.Fatalf("expected returning menu, reply=%#v err=%v", reply, err)
	}
	user, _ := users.GetUserByPhone(ctx, phone)
	if user.VerificationTier != models.VerificationTierMedium {
		t.Fatalf("expected medium risk, got %s", user.VerificationTier)
	}
}

func TestWhatsAppBotSkipVerificationFlow(t *testing.T) {
	ctx := context.Background()
	users := newBotUserRepository()
	phone := "+919876543227"
	bot := newTestWhatsAppBot(users, nil, newRecordingWhatsAppSender())
	completeProfileSetup(t, ctx, bot, phone)

	reply, err := bot.HandleIncomingMessage(ctx, IncomingWhatsAppMessage{PhoneNumber: phone, Text: "C"})
	if err != nil || reply.State != BotStateAwaitingSkipConfirm {
		t.Fatalf("expected skip confirm, reply=%#v err=%v", reply, err)
	}
	reply, err = bot.HandleIncomingMessage(ctx, IncomingWhatsAppMessage{PhoneNumber: phone, Text: "YES"})
	if err != nil || reply.State != BotStateReturningMenu {
		t.Fatalf("expected returning menu, reply=%#v err=%v", reply, err)
	}
	user, _ := users.GetUserByPhone(ctx, phone)
	if user.VerificationTier != models.VerificationTierHigh {
		t.Fatalf("expected high risk, got %s", user.VerificationTier)
	}
}

func TestWhatsAppBotBrowseJobsAndApplyFlow(t *testing.T) {
	ctx := context.Background()
	users := newBotUserRepository()
	user := users.mustCreate("+919876543228", "Job Worker")
	apps := newBotApplicationService(nil)
	bot := newTestWhatsAppBotWithDeps(users, apps, newBotJobService(defaultBotJobs()), newBotIdentityService(users), newRecordingWhatsAppSender())

	if _, err := bot.HandleIncomingMessage(ctx, IncomingWhatsAppMessage{PhoneNumber: user.PhoneNumber, Text: "menu"}); err != nil {
		t.Fatalf("menu: %v", err)
	}
	reply, err := bot.HandleIncomingMessage(ctx, IncomingWhatsAppMessage{PhoneNumber: user.PhoneNumber, Text: "3"})
	if err != nil || reply.State != BotStateAwaitingJobID || !strings.Contains(reply.Message, "job-1") {
		t.Fatalf("expected job list, reply=%#v err=%v", reply, err)
	}
	reply, err = bot.HandleIncomingMessage(ctx, IncomingWhatsAppMessage{PhoneNumber: user.PhoneNumber, Text: "bad-job"})
	if err != nil || reply.State != BotStateAwaitingJobID || !strings.Contains(reply.Message, "Invalid job ID") {
		t.Fatalf("expected invalid job fallback, reply=%#v err=%v", reply, err)
	}
	reply, err = bot.HandleIncomingMessage(ctx, IncomingWhatsAppMessage{PhoneNumber: user.PhoneNumber, Text: "job-1"})
	if err != nil || reply.State != BotStateReturningMenu || !strings.Contains(reply.Message, "Application submitted") {
		t.Fatalf("expected application confirmation, reply=%#v err=%v", reply, err)
	}
	if len(apps.applications) != 1 {
		t.Fatalf("expected one application, got %d", len(apps.applications))
	}
}

func TestWhatsAppBotInvalidInputHandling(t *testing.T) {
	ctx := context.Background()
	users := newBotUserRepository()
	phone := "+919876543229"
	bot := newTestWhatsAppBot(users, nil, newRecordingWhatsAppSender())
	completeProfileSetup(t, ctx, bot, phone)

	reply, err := bot.HandleIncomingMessage(ctx, IncomingWhatsAppMessage{PhoneNumber: phone, Text: "Z"})
	if err != nil || reply.State != BotStateAwaitingVerificationOption || !strings.Contains(reply.Message, "A, B, or C") {
		t.Fatalf("expected invalid verification fallback, reply=%#v err=%v", reply, err)
	}
	reply, err = bot.HandleIncomingMessage(ctx, IncomingWhatsAppMessage{PhoneNumber: phone, Text: "A"})
	if err != nil || reply.State != BotStateAwaitingAadhaarNumber {
		t.Fatalf("expected aadhaar state, reply=%#v err=%v", reply, err)
	}
	reply, err = bot.HandleIncomingMessage(ctx, IncomingWhatsAppMessage{PhoneNumber: phone, Text: "123"})
	if err != nil || reply.State != BotStateAwaitingAadhaarNumber || !strings.Contains(reply.Message, "12 digits") {
		t.Fatalf("expected invalid aadhaar fallback, reply=%#v err=%v", reply, err)
	}
}

func newTestWhatsAppBot(users *botUserRepository, apps *botApplicationService, sender *recordingWhatsAppSender) WhatsAppBotService {
	if apps == nil {
		apps = newBotApplicationService(nil)
	}
	store := newFakeRedisStore()
	return NewWhatsAppBotService(users, apps, newBotJobService(defaultBotJobs()), newBotIdentityService(users), newBotReferralService(), NewRedisConversationStateService(store), NewRedisStatusOTPService(store, "pepper"), sender)
}

func newTestWhatsAppBotWithDeps(users *botUserRepository, apps *botApplicationService, jobs *botJobService, identity *botIdentityService, sender *recordingWhatsAppSender) WhatsAppBotService {
	if apps == nil {
		apps = newBotApplicationService(nil)
	}
	if jobs == nil {
		jobs = newBotJobService(defaultBotJobs())
	}
	store := newFakeRedisStore()
	return NewWhatsAppBotService(users, apps, jobs, identity, newBotReferralService(), NewRedisConversationStateService(store), NewRedisStatusOTPService(store, "pepper"), sender)
}

func completeProfileSetup(t *testing.T, ctx context.Context, bot WhatsAppBotService, phone string) {
	t.Helper()
	steps := []string{"hi", "Ravi Kumar", "1", "Machine Operator", "Pune"}
	for _, step := range steps {
		if _, err := bot.HandleIncomingMessage(ctx, IncomingWhatsAppMessage{PhoneNumber: phone, Text: step}); err != nil {
			t.Fatalf("profile step %q: %v", step, err)
		}
	}
}

func mustRaw(data botStateData) json.RawMessage {
	encoded, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}
	return encoded
}

type recordingWhatsAppSender struct {
	messages []string
}

func newRecordingWhatsAppSender() *recordingWhatsAppSender {
	return &recordingWhatsAppSender{}
}

func (s *recordingWhatsAppSender) SendMessage(ctx context.Context, phoneNumber, message string) error {
	s.messages = append(s.messages, message)
	return nil
}

type botUserRepository struct {
	byID    map[string]*models.User
	byPhone map[string]*models.User
	nextID  int
}

func newBotUserRepository() *botUserRepository {
	return &botUserRepository{byID: map[string]*models.User{}, byPhone: map[string]*models.User{}}
}

func (r *botUserRepository) mustCreate(phoneNumber, fullName string) *models.User {
	user, err := r.CreateUser(context.Background(), &models.User{
		PhoneNumber:        phoneNumber,
		FullName:           fullName,
		LanguagePreference: "en",
		VerificationTier:   models.VerificationTierHigh,
		ReferralCode:       referralCodeFromPhone(phoneNumber),
		IsActive:           true,
	})
	if err != nil {
		panic(err)
	}
	return user
}

func (r *botUserRepository) CreateUser(ctx context.Context, user *models.User) (*models.User, error) {
	r.nextID++
	copyUser := *user
	copyUser.ID = fmt.Sprintf("worker-test-%d", r.nextID)
	copyUser.CreatedAt = time.Now()
	copyUser.UpdatedAt = copyUser.CreatedAt
	r.byID[copyUser.ID] = &copyUser
	r.byPhone[copyUser.PhoneNumber] = &copyUser
	return &copyUser, nil
}

func (r *botUserRepository) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	user, ok := r.byID[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	copyUser := *user
	return &copyUser, nil
}

func (r *botUserRepository) GetUserByPhone(ctx context.Context, phoneNumber string) (*models.User, error) {
	user, ok := r.byPhone[phoneNumber]
	if !ok {
		return nil, repository.ErrNotFound
	}
	copyUser := *user
	return &copyUser, nil
}

func (r *botUserRepository) UpdateUserProfile(ctx context.Context, user *models.User) (*models.User, error) {
	if _, ok := r.byID[user.ID]; !ok {
		return nil, repository.ErrNotFound
	}
	copyUser := *user
	copyUser.UpdatedAt = time.Now()
	r.byID[user.ID] = &copyUser
	r.byPhone[user.PhoneNumber] = &copyUser
	return &copyUser, nil
}

func (r *botUserRepository) UpdateUserVerificationTier(ctx context.Context, id string, tier models.VerificationTier) (*models.User, error) {
	user, err := r.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}
	user.VerificationTier = tier
	return r.UpdateUserProfile(ctx, user)
}

func (r *botUserRepository) GetUserByReferralCode(ctx context.Context, referralCode string) (*models.User, error) {
	for _, user := range r.byID {
		if user.ReferralCode == referralCode {
			copyUser := *user
			return &copyUser, nil
		}
	}
	return nil, repository.ErrNotFound
}

type botApplicationService struct {
	applications []models.Application
	jobs         map[string]models.Job
}

func newBotApplicationService(applications []models.Application) *botApplicationService {
	jobs := map[string]models.Job{}
	for _, job := range defaultBotJobs() {
		jobs[job.ID] = job
	}
	return &botApplicationService{applications: applications, jobs: jobs}
}

func (r *botApplicationService) CreateApplication(ctx context.Context, input CreateApplicationInput) (*models.Application, error) {
	job, ok := r.jobs[input.JobID]
	if !ok {
		return nil, repository.ErrNotFound
	}
	application := models.Application{
		ID:         fmt.Sprintf("application-%d", len(r.applications)+1),
		UserID:     input.UserID,
		JobID:      input.JobID,
		EmployerID: job.EmployerID,
		Status:     models.ApplicationStatusApplied,
		Source:     input.Source,
	}
	r.applications = append(r.applications, application)
	return &application, nil
}

func (r *botApplicationService) GetApplicationByID(ctx context.Context, id string) (*models.Application, error) {
	for _, application := range r.applications {
		if application.ID == id {
			return &application, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (r *botApplicationService) ListApplicationsByUser(ctx context.Context, userID string, limit, offset int) ([]models.Application, error) {
	var result []models.Application
	for _, application := range r.applications {
		if application.UserID == userID {
			result = append(result, application)
		}
	}
	return result, nil
}

type botJobService struct {
	jobs []models.Job
}

func newBotJobService(jobs []models.Job) *botJobService {
	return &botJobService{jobs: jobs}
}

func (s *botJobService) GetJobByID(ctx context.Context, id string) (*models.Job, error) {
	for _, job := range s.jobs {
		if job.ID == id {
			return &job, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (s *botJobService) ListActiveJobs(ctx context.Context, limit, offset int) ([]models.Job, error) {
	return s.jobs, nil
}

func defaultBotJobs() []models.Job {
	return []models.Job{{
		ID:            "job-1",
		EmployerID:    "employer-1",
		Title:         "Machine Operator",
		Role:          "Machine Operator",
		LocationCity:  "Pune",
		LocationState: "Maharashtra",
		ShiftSchedule: "Day shift",
		IsActive:      true,
	}}
}

type botIdentityService struct {
	users *botUserRepository
}

func newBotIdentityService(users *botUserRepository) *botIdentityService {
	return &botIdentityService{users: users}
}

func (s *botIdentityService) StartAadhaarOTP(ctx context.Context, userID, aadhaarNumber string, consentGiven bool) (*models.WorkerIdentityVerification, error) {
	ref := "mock-aadhaar-otp-transaction"
	last4 := aadhaarNumber[len(aadhaarNumber)-4:]
	return &models.WorkerIdentityVerification{
		ID:                  "verification-1",
		UserID:              userID,
		Method:              models.IdentityVerificationMethodAadhaarOTP,
		Status:              models.IdentityVerificationStatusOTPSent,
		AadhaarLast4:        &last4,
		AadhaarReferenceKey: &ref,
		ConsentGiven:        consentGiven,
	}, nil
}

func (s *botIdentityService) VerifyAadhaarOTP(ctx context.Context, userID, transactionID, otp string) (*models.WorkerIdentityVerification, error) {
	user, _ := s.users.GetUserByID(ctx, userID)
	user.VerificationTier = models.VerificationTierLow
	_, _ = s.users.UpdateUserProfile(ctx, user)
	return &models.WorkerIdentityVerification{
		ID:     "verification-1",
		UserID: userID,
		Method: models.IdentityVerificationMethodAadhaarOTP,
		Status: models.IdentityVerificationStatusVerified,
	}, nil
}

func (s *botIdentityService) MarkDocumentUploaded(ctx context.Context, userID, documentRef string) (*models.WorkerIdentityVerification, error) {
	user, _ := s.users.GetUserByID(ctx, userID)
	user.VerificationTier = models.VerificationTierMedium
	_, _ = s.users.UpdateUserProfile(ctx, user)
	return &models.WorkerIdentityVerification{
		ID:          "verification-1",
		UserID:      userID,
		Method:      models.IdentityVerificationMethodDocumentUpload,
		Status:      models.IdentityVerificationStatusDocumentUploaded,
		DocumentRef: &documentRef,
	}, nil
}

func (s *botIdentityService) MarkSkipped(ctx context.Context, userID, reason string) (*models.WorkerIdentityVerification, error) {
	user, _ := s.users.GetUserByID(ctx, userID)
	user.VerificationTier = models.VerificationTierHigh
	_, _ = s.users.UpdateUserProfile(ctx, user)
	return &models.WorkerIdentityVerification{
		ID:           "verification-1",
		UserID:       userID,
		Method:       models.IdentityVerificationMethodSkipped,
		Status:       models.IdentityVerificationStatusSkipped,
		FailedReason: &reason,
	}, nil
}

func (s *botIdentityService) GetLatest(ctx context.Context, userID string) (*models.WorkerIdentityVerification, error) {
	return nil, repository.ErrNotFound
}

type botReferralService struct{}

func newBotReferralService() *botReferralService {
	return &botReferralService{}
}

func (s *botReferralService) RegisterReferral(ctx context.Context, referredUser *models.User) (*models.Referral, error) {
	if referredUser.ReferredByCode == nil {
		return nil, nil
	}
	return &models.Referral{
		ID:             "referral-1",
		ReferrerUserID: "referrer-1",
		ReferredUserID: &referredUser.ID,
		ReferralCode:   *referredUser.ReferredByCode,
	}, nil
}

func (s *botReferralService) CompleteOnboarding(ctx context.Context, referredUserID string) (*models.ReferralTransaction, error) {
	return &models.ReferralTransaction{
		ID:          "referral-transaction-1",
		ReferralID:  "referral-1",
		UserID:      "referrer-1",
		AmountPaise: referralCashbackAmountPaise,
		Currency:    "INR",
		Status:      "Pending",
	}, nil
}

func (s *botReferralService) GetWorkerReferral(ctx context.Context, workerID string) (*models.User, error) {
	return nil, repository.ErrNotFound
}

func (s *botReferralService) ListReferrals(ctx context.Context, workerID string, limit, offset int) ([]models.Referral, error) {
	return nil, nil
}

func (s *botReferralService) ListTransactions(ctx context.Context, workerID string, limit, offset int) ([]models.ReferralTransaction, error) {
	return nil, nil
}

func (s *botReferralService) ProcessPendingPayouts(ctx context.Context, limit int) (ReferralPayoutProcessResult, error) {
	return ReferralPayoutProcessResult{}, nil
}
