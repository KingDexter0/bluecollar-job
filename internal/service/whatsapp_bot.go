package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"bluecollarjob/internal/models"
	"bluecollarjob/internal/repository"
)

const (
	BotStateAwaitingName               = "awaiting_name"
	BotStateAwaitingLanguage           = "awaiting_language"
	BotStateAwaitingTargetRole         = "awaiting_target_role"
	BotStateAwaitingPreferredZone      = "awaiting_preferred_zone"
	BotStateAwaitingVerificationOption = "awaiting_verification_option"
	BotStateAwaitingAadhaarNumber      = "awaiting_aadhaar_number"
	BotStateAwaitingAadhaarConsent     = "awaiting_aadhaar_consent"
	BotStateAwaitingAadhaarOTP         = "awaiting_aadhaar_otp"
	BotStateAwaitingDocumentUpload     = "awaiting_document_upload"
	BotStateAwaitingSkipConfirm        = "awaiting_skip_confirm"
	BotStateReturningMenu              = "returning_menu"
	BotStateAwaitingStatusOTP          = "awaiting_status_otp"
	BotStateBrowsingJobs               = "browsing_jobs"
	BotStateAwaitingJobID              = "awaiting_job_id"
)

var aadhaarChatPattern = regexp.MustCompile(`^[0-9]{12}$`)

type WhatsAppBotService interface {
	HandleIncomingMessage(ctx context.Context, message IncomingWhatsAppMessage) (*BotReply, error)
}

type IncomingWhatsAppMessage struct {
	PhoneNumber string
	Text        string
	MessageType string
	MediaRef    *string
}

type BotReply struct {
	PhoneNumber string `json:"phone_number"`
	Message     string `json:"message"`
	State       string `json:"state"`
}

type whatsappBotService struct {
	users        repository.UserRepository
	applications ApplicationService
	jobs         JobService
	identity     IdentityVerificationService
	referrals    ReferralService
	states       ConversationStateService
	statusOTPs   StatusOTPService
	sender       WhatsAppSender
}

type botStateData struct {
	WorkerID      string `json:"worker_id,omitempty"`
	FullName      string `json:"full_name,omitempty"`
	Language      string `json:"language,omitempty"`
	TargetRole    string `json:"target_role,omitempty"`
	PreferredZone string `json:"preferred_zone,omitempty"`
	TransactionID string `json:"transaction_id,omitempty"`
	AadhaarLast4  string `json:"aadhaar_last4,omitempty"`
}

func NewWhatsAppBotService(users repository.UserRepository, applications ApplicationService, jobs JobService, identity IdentityVerificationService, referrals ReferralService, states ConversationStateService, statusOTPs StatusOTPService, sender WhatsAppSender) WhatsAppBotService {
	return &whatsappBotService{
		users:        users,
		applications: applications,
		jobs:         jobs,
		identity:     identity,
		referrals:    referrals,
		states:       states,
		statusOTPs:   statusOTPs,
		sender:       sender,
	}
}

func (s *whatsappBotService) HandleIncomingMessage(ctx context.Context, message IncomingWhatsAppMessage) (*BotReply, error) {
	message.PhoneNumber = strings.TrimSpace(message.PhoneNumber)
	message.Text = strings.TrimSpace(message.Text)
	if message.PhoneNumber == "" {
		return nil, fmt.Errorf("%w: phone number is required", ErrInvalidInput)
	}

	user, err := s.users.GetUserByPhone(ctx, message.PhoneNumber)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return s.startNewUserFlow(ctx, message.PhoneNumber, message.Text)
		}
		return nil, err
	}
	if isReferralCommand(message.Text) {
		return s.replyAndSetState(ctx, user.PhoneNumber, BotStateReturningMenu, botStateData{WorkerID: user.ID, Language: normalizeLanguage(user.LanguagePreference)}, fmt.Sprintf(localizedReply(user.LanguagePreference, "show_referral_code"), user.ReferralCode)+"\n\n"+localizedReply(user.LanguagePreference, "returning_menu"))
	}

	state, err := s.states.GetState(ctx, message.PhoneNumber)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return s.showReturningMenu(ctx, user)
		}
		return nil, err
	}

	data := decodeBotStateData(state.Data)
	if data.WorkerID == "" {
		data.WorkerID = user.ID
	}

	switch state.State {
	case BotStateAwaitingName:
		return s.handleName(ctx, user, message.Text, data)
	case BotStateAwaitingLanguage:
		return s.handleLanguage(ctx, user, message.Text, data)
	case BotStateAwaitingTargetRole:
		return s.handleTargetRole(ctx, user, message.Text, data)
	case BotStateAwaitingPreferredZone:
		return s.handlePreferredZone(ctx, user, message.Text, data)
	case BotStateAwaitingVerificationOption:
		return s.handleVerificationOption(ctx, user, message.Text, data)
	case BotStateAwaitingAadhaarNumber:
		return s.handleAadhaarNumber(ctx, user, message.Text, data)
	case BotStateAwaitingAadhaarConsent:
		return s.handleAadhaarConsent(ctx, user, message.Text, data)
	case BotStateAwaitingAadhaarOTP:
		return s.handleAadhaarOTP(ctx, user, message.Text, data)
	case BotStateAwaitingDocumentUpload:
		return s.handleDocumentUpload(ctx, user, message, data)
	case BotStateAwaitingSkipConfirm:
		return s.handleSkipConfirm(ctx, user, message.Text, data)
	case BotStateReturningMenu:
		return s.handleReturningMenu(ctx, user, message.Text, data)
	case BotStateAwaitingStatusOTP:
		return s.handleStatusOTP(ctx, user, message.Text, data)
	case BotStateBrowsingJobs, BotStateAwaitingJobID:
		return s.handleJobID(ctx, user, message.Text, data)
	default:
		return s.showReturningMenu(ctx, user)
	}
}

func (s *whatsappBotService) startNewUserFlow(ctx context.Context, phoneNumber, text string) (*BotReply, error) {
	referredByCode := parseReferralCodeCommand(text)
	user, err := s.users.CreateUser(ctx, &models.User{
		PhoneNumber:        phoneNumber,
		FullName:           "WhatsApp Worker " + lastDigits(phoneNumber, 4),
		LanguagePreference: "en",
		VerificationTier:   models.VerificationTierHigh,
		ReferralCode:       referralCodeFromPhone(phoneNumber),
		ReferredByCode:     stringPtrFromValue(referredByCode),
		IsActive:           true,
	})
	if err != nil {
		return nil, err
	}
	if s.referrals != nil {
		if _, err := s.referrals.RegisterReferral(ctx, user); err != nil {
			return nil, err
		}
	}

	return s.replyAndSetState(ctx, phoneNumber, BotStateAwaitingName, botStateData{WorkerID: user.ID, Language: "en"}, fmt.Sprintf(localizedReply("en", "new_user_greeting"), user.ReferralCode))
}

func (s *whatsappBotService) showReturningMenu(ctx context.Context, user *models.User) (*BotReply, error) {
	data := botStateData{WorkerID: user.ID, Language: normalizeLanguage(user.LanguagePreference)}
	return s.replyAndSetState(ctx, user.PhoneNumber, BotStateReturningMenu, data, localizedReply(data.Language, "returning_menu"))
}

func (s *whatsappBotService) handleName(ctx context.Context, user *models.User, text string, data botStateData) (*BotReply, error) {
	if text == "" {
		return s.replyAndSetState(ctx, user.PhoneNumber, BotStateAwaitingName, data, localizedReply(data.Language, "ask_name"))
	}
	data.FullName = text
	user.FullName = text
	if _, err := s.users.UpdateUserProfile(ctx, user); err != nil {
		return nil, err
	}
	return s.replyAndSetState(ctx, user.PhoneNumber, BotStateAwaitingLanguage, data, localizedReply(data.Language, "ask_language"))
}

func (s *whatsappBotService) handleLanguage(ctx context.Context, user *models.User, text string, data botStateData) (*BotReply, error) {
	language, ok := parseLanguage(text)
	if !ok {
		return s.replyAndSetState(ctx, user.PhoneNumber, BotStateAwaitingLanguage, data, localizedReply(data.Language, "ask_language"))
	}
	data.Language = language
	user.LanguagePreference = language
	user.FullName = fallbackName(user, data)
	if _, err := s.users.UpdateUserProfile(ctx, user); err != nil {
		return nil, err
	}
	return s.replyAndSetState(ctx, user.PhoneNumber, BotStateAwaitingTargetRole, data, localizedReply(language, "ask_target_role"))
}

func (s *whatsappBotService) handleTargetRole(ctx context.Context, user *models.User, text string, data botStateData) (*BotReply, error) {
	if text == "" {
		return s.replyAndSetState(ctx, user.PhoneNumber, BotStateAwaitingTargetRole, data, localizedReply(data.Language, "ask_target_role"))
	}
	data.TargetRole = text
	user.FullName = fallbackName(user, data)
	user.LanguagePreference = data.Language
	user.TargetRole = &data.TargetRole
	if _, err := s.users.UpdateUserProfile(ctx, user); err != nil {
		return nil, err
	}
	return s.replyAndSetState(ctx, user.PhoneNumber, BotStateAwaitingPreferredZone, data, localizedReply(data.Language, "ask_preferred_zone"))
}

func (s *whatsappBotService) handlePreferredZone(ctx context.Context, user *models.User, text string, data botStateData) (*BotReply, error) {
	if text == "" {
		return s.replyAndSetState(ctx, user.PhoneNumber, BotStateAwaitingPreferredZone, data, localizedReply(data.Language, "ask_preferred_zone"))
	}
	data.PreferredZone = text
	user.FullName = fallbackName(user, data)
	user.LanguagePreference = data.Language
	user.TargetRole = stringPtrFromValue(data.TargetRole)
	user.PreferredZone = &data.PreferredZone
	if _, err := s.users.UpdateUserProfile(ctx, user); err != nil {
		return nil, err
	}
	return s.replyAndSetState(ctx, user.PhoneNumber, BotStateAwaitingVerificationOption, data, localizedReply(data.Language, "ask_verification_option"))
}

func (s *whatsappBotService) handleVerificationOption(ctx context.Context, user *models.User, text string, data botStateData) (*BotReply, error) {
	choice := strings.ToUpper(strings.TrimSpace(text))
	switch choice {
	case "A", "1":
		return s.replyAndSetState(ctx, user.PhoneNumber, BotStateAwaitingAadhaarNumber, data, localizedReply(data.Language, "ask_aadhaar_number"))
	case "B", "2":
		return s.replyAndSetState(ctx, user.PhoneNumber, BotStateAwaitingDocumentUpload, data, localizedReply(data.Language, "ask_document_upload"))
	case "C", "3":
		return s.replyAndSetState(ctx, user.PhoneNumber, BotStateAwaitingSkipConfirm, data, localizedReply(data.Language, "confirm_skip_verification"))
	default:
		return s.replyAndSetState(ctx, user.PhoneNumber, BotStateAwaitingVerificationOption, data, localizedReply(data.Language, "invalid_verification_option")+"\n\n"+localizedReply(data.Language, "ask_verification_option"))
	}
}

func (s *whatsappBotService) handleAadhaarNumber(ctx context.Context, user *models.User, text string, data botStateData) (*BotReply, error) {
	aadhaarNumber := strings.TrimSpace(text)
	if !aadhaarChatPattern.MatchString(aadhaarNumber) {
		return s.replyAndSetState(ctx, user.PhoneNumber, BotStateAwaitingAadhaarNumber, data, localizedReply(data.Language, "invalid_aadhaar")+"\n\n"+localizedReply(data.Language, "ask_aadhaar_number"))
	}
	data.AadhaarLast4 = aadhaarNumber[len(aadhaarNumber)-4:]
	return s.replyAndSetState(ctx, user.PhoneNumber, BotStateAwaitingAadhaarConsent, data, localizedReply(data.Language, "ask_aadhaar_consent"))
}

func (s *whatsappBotService) handleAadhaarConsent(ctx context.Context, user *models.User, text string, data botStateData) (*BotReply, error) {
	parts := strings.Fields(strings.TrimSpace(text))
	if len(parts) < 2 || !isYes(parts[0]) || !aadhaarChatPattern.MatchString(parts[1]) {
		return s.replyAndSetState(ctx, user.PhoneNumber, BotStateAwaitingAadhaarConsent, data, localizedReply(data.Language, "ask_aadhaar_consent"))
	}
	aadhaarNumber := parts[1]
	if data.AadhaarLast4 != "" && !strings.HasSuffix(aadhaarNumber, data.AadhaarLast4) {
		return s.replyAndSetState(ctx, user.PhoneNumber, BotStateAwaitingAadhaarConsent, data, localizedReply(data.Language, "invalid_aadhaar")+"\n\n"+localizedReply(data.Language, "ask_aadhaar_consent"))
	}
	verification, err := s.identity.StartAadhaarOTP(ctx, user.ID, aadhaarNumber, true)
	if err != nil {
		return nil, err
	}
	if verification.AadhaarReferenceKey != nil {
		data.TransactionID = *verification.AadhaarReferenceKey
	}
	return s.replyAndSetState(ctx, user.PhoneNumber, BotStateAwaitingAadhaarOTP, data, localizedReply(data.Language, "ask_aadhaar_otp"))
}

func (s *whatsappBotService) handleAadhaarOTP(ctx context.Context, user *models.User, text string, data botStateData) (*BotReply, error) {
	otp := strings.TrimSpace(text)
	if otp == "" {
		return s.replyAndSetState(ctx, user.PhoneNumber, BotStateAwaitingAadhaarOTP, data, localizedReply(data.Language, "invalid_otp"))
	}
	if _, err := s.identity.VerifyAadhaarOTP(ctx, user.ID, data.TransactionID, otp); err != nil {
		return s.replyAndSetState(ctx, user.PhoneNumber, BotStateAwaitingAadhaarOTP, data, localizedReply(data.Language, "invalid_otp"))
	}
	data.TransactionID = ""
	data.AadhaarLast4 = ""
	return s.replyAndSetState(ctx, user.PhoneNumber, BotStateReturningMenu, data, localizedReply(data.Language, "aadhaar_verified")+"\n"+fmt.Sprintf(localizedReply(data.Language, "show_referral_code"), user.ReferralCode)+"\n\n"+localizedReply(data.Language, "returning_menu"))
}

func (s *whatsappBotService) handleDocumentUpload(ctx context.Context, user *models.User, message IncomingWhatsAppMessage, data botStateData) (*BotReply, error) {
	documentRef := ""
	if message.MediaRef != nil {
		documentRef = strings.TrimSpace(*message.MediaRef)
	}
	if documentRef == "" {
		return s.replyAndSetState(ctx, user.PhoneNumber, BotStateAwaitingDocumentUpload, data, localizedReply(data.Language, "missing_document_upload"))
	}
	if _, err := s.identity.MarkDocumentUploaded(ctx, user.ID, documentRef); err != nil {
		return nil, err
	}
	return s.replyAndSetState(ctx, user.PhoneNumber, BotStateReturningMenu, data, localizedReply(data.Language, "document_uploaded")+"\n"+fmt.Sprintf(localizedReply(data.Language, "show_referral_code"), user.ReferralCode)+"\n\n"+localizedReply(data.Language, "returning_menu"))
}

func (s *whatsappBotService) handleSkipConfirm(ctx context.Context, user *models.User, text string, data botStateData) (*BotReply, error) {
	if !isYes(text) {
		return s.replyAndSetState(ctx, user.PhoneNumber, BotStateAwaitingVerificationOption, data, localizedReply(data.Language, "ask_verification_option"))
	}
	if _, err := s.identity.MarkSkipped(ctx, user.ID, "Worker skipped verification from WhatsApp onboarding."); err != nil {
		return nil, err
	}
	return s.replyAndSetState(ctx, user.PhoneNumber, BotStateReturningMenu, data, localizedReply(data.Language, "verification_skip_selected")+"\n"+fmt.Sprintf(localizedReply(data.Language, "show_referral_code"), user.ReferralCode)+"\n\n"+localizedReply(data.Language, "returning_menu"))
}

func (s *whatsappBotService) handleReturningMenu(ctx context.Context, user *models.User, text string, data botStateData) (*BotReply, error) {
	switch strings.TrimSpace(text) {
	case "1":
		otp, err := s.statusOTPs.Generate(ctx, user.PhoneNumber)
		if err != nil {
			return nil, err
		}
		data.TransactionID = otp.TransactionID
		message := localizedReply(data.Language, "status_otp_sent")
		if otp.OTPForLocalDev != "" {
			message += "\nLocal dev OTP: " + otp.OTPForLocalDev
		}
		return s.replyAndSetState(ctx, user.PhoneNumber, BotStateAwaitingStatusOTP, data, message)
	case "2":
		return s.replyAndSetState(ctx, user.PhoneNumber, BotStateAwaitingTargetRole, data, localizedReply(data.Language, "ask_target_role"))
	case "3":
		return s.showActiveJobs(ctx, user, data)
	case "4":
		return s.replyAndSetState(ctx, user.PhoneNumber, BotStateAwaitingJobID, data, localizedReply(data.Language, "ask_job_id"))
	default:
		return s.replyAndSetState(ctx, user.PhoneNumber, BotStateReturningMenu, data, localizedReply(data.Language, "invalid_menu_option")+"\n\n"+localizedReply(data.Language, "returning_menu"))
	}
}

func (s *whatsappBotService) showActiveJobs(ctx context.Context, user *models.User, data botStateData) (*BotReply, error) {
	jobs, err := s.jobs.ListActiveJobs(ctx, 10, 0)
	if err != nil {
		return nil, err
	}
	return s.replyAndSetState(ctx, user.PhoneNumber, BotStateAwaitingJobID, data, buildJobsReply(data.Language, jobs)+"\n\n"+localizedReply(data.Language, "ask_job_id"))
}

func (s *whatsappBotService) handleJobID(ctx context.Context, user *models.User, text string, data botStateData) (*BotReply, error) {
	jobID := strings.TrimSpace(text)
	if jobID == "" || strings.EqualFold(jobID, "browse") {
		return s.showActiveJobs(ctx, user, data)
	}
	application, err := s.applications.CreateApplication(ctx, CreateApplicationInput{
		UserID: user.ID,
		JobID:  jobID,
		Source: "whatsapp",
	})
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return s.replyAndSetState(ctx, user.PhoneNumber, BotStateAwaitingJobID, data, localizedReply(data.Language, "invalid_job_id")+"\n\n"+localizedReply(data.Language, "ask_job_id"))
		}
		return nil, err
	}
	return s.replyAndSetState(ctx, user.PhoneNumber, BotStateReturningMenu, data, fmt.Sprintf(localizedReply(data.Language, "application_created"), application.ID)+"\n\n"+localizedReply(data.Language, "returning_menu"))
}

func (s *whatsappBotService) handleStatusOTP(ctx context.Context, user *models.User, text string, data botStateData) (*BotReply, error) {
	if err := s.statusOTPs.Verify(ctx, user.PhoneNumber, data.TransactionID, text); err != nil {
		return s.replyAndSetState(ctx, user.PhoneNumber, BotStateAwaitingStatusOTP, data, localizedReply(data.Language, "invalid_otp"))
	}

	applications, err := s.applications.ListApplicationsByUser(ctx, user.ID, 10, 0)
	if err != nil {
		return nil, err
	}
	message := buildApplicationStatusReply(data.Language, applications)
	data.TransactionID = ""
	return s.replyAndSetState(ctx, user.PhoneNumber, BotStateReturningMenu, data, message+"\n\n"+localizedReply(data.Language, "returning_menu"))
}

func (s *whatsappBotService) replyAndSetState(ctx context.Context, phoneNumber, state string, data botStateData, message string) (*BotReply, error) {
	encoded, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	if _, err := s.states.SetState(ctx, phoneNumber, state, encoded, 24*time.Hour); err != nil {
		return nil, err
	}
	if err := s.sender.SendMessage(ctx, phoneNumber, message); err != nil {
		return nil, err
	}
	return &BotReply{PhoneNumber: phoneNumber, Message: message, State: state}, nil
}

func decodeBotStateData(raw json.RawMessage) botStateData {
	var data botStateData
	_ = json.Unmarshal(raw, &data)
	if data.Language == "" {
		data.Language = "en"
	}
	return data
}

func parseLanguage(text string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(text)) {
	case "1", "english", "en":
		return "en", true
	case "2", "hindi", "hi", "हिंदी":
		return "hi", true
	case "3", "marathi", "mr", "मराठी":
		return "mr", true
	case "4", "telugu", "te", "తెలుగు":
		return "te", true
	default:
		return "", false
	}
}

func normalizeLanguage(language string) string {
	if parsed, ok := parseLanguage(language); ok {
		return parsed
	}
	return "en"
}

func localizedReply(language, key string) string {
	language = normalizeLanguage(language)
	replies := map[string]map[string]string{
		"en": {
			"new_user_greeting":              "Welcome to BlueCollarJob. Your referral code is %s. Please tell us your full name.",
			"ask_name":                       "Please enter your full name.",
			"ask_language":                   "Choose language:\n1. English\n2. Hindi\n3. Marathi\n4. Telugu",
			"ask_target_role":                "What job role are you looking for?",
			"ask_preferred_zone":             "Which work location or zone do you prefer?",
			"ask_verification_option":        "Choose verification option:\nA. Aadhaar OTP\nB. Document Upload\nC. Skip Verification",
			"returning_menu":                 "Menu:\n1. Check Application Status\n2. Update Profile\n3. Browse Jobs\n4. Apply to Job",
			"status_otp_sent":                "We sent a 6-digit OTP. Please reply with the OTP to view application status.",
			"invalid_otp":                    "Invalid OTP. Please try again.",
			"no_applications":                "No applications found yet.",
			"verification_aadhaar_selected":  "Aadhaar OTP verification will start from the identity verification API.",
			"verification_document_selected": "Document upload verification will continue from the identity verification API.",
			"verification_skip_selected":     "Verification skipped for now. You can complete it later.",
			"invalid_verification_option":    "Please choose A, B, or C.",
			"ask_aadhaar_number":             "Please enter your 12-digit Aadhaar number.",
			"invalid_aadhaar":                "Invalid Aadhaar format. Aadhaar must be 12 digits.",
			"ask_aadhaar_consent":            "For consent, reply YES followed by your 12-digit Aadhaar number. Example: YES 123456789012",
			"ask_aadhaar_otp":                "Mock Aadhaar OTP sent. Please enter the OTP.",
			"aadhaar_verified":               "Aadhaar OTP verified. Your risk tier is now Low.",
			"ask_document_upload":            "Please upload your Aadhaar/document photo.",
			"missing_document_upload":        "Please upload a document/photo to continue.",
			"document_uploaded":              "Document uploaded. Your risk tier is now Medium.",
			"confirm_skip_verification":      "Reply YES to skip verification for now.",
			"invalid_menu_option":            "Invalid menu option.",
			"ask_job_id":                     "Send the job ID you want to apply for, or type browse to list jobs again.",
			"invalid_job_id":                 "Invalid job ID. Please send a valid job ID.",
			"no_jobs":                        "No active jobs found right now.",
			"application_created":            "Application submitted successfully. Application ID: %s",
			"show_referral_code":             "Your referral code is %s. Share it and earn Rs 100 cashback after your referral completes onboarding.",
		},
		"hi": {
			"new_user_greeting":       "BlueCollarJob mein swagat hai. Kripya apna poora naam batayein.",
			"ask_language":            "Bhasha chunein:\n1. English\n2. Hindi\n3. Marathi\n4. Telugu",
			"returning_menu":          "Menu:\n1. Application Status\n2. Profile Update\n3. Jobs Dekhein\n4. Job Apply Karein",
			"status_otp_sent":         "6 digit OTP bheja gaya hai. Status dekhne ke liye OTP reply karein.",
			"invalid_otp":             "OTP galat hai. Dobara try karein.",
			"ask_target_role":         "Aap kaunsa job role dhoond rahe hain?",
			"ask_preferred_zone":      "Aap kaunsi location ya zone prefer karte hain?",
			"ask_verification_option": "Verification option chunein:\nA. Aadhaar OTP\nB. Document Upload\nC. Skip Verification",
			"invalid_menu_option":     "Invalid menu option.",
		},
		"mr": {
			"new_user_greeting":       "BlueCollarJob madhye swagat aahe. Krupaya tumche purna nav sanga.",
			"ask_language":            "Bhasha nivda:\n1. English\n2. Hindi\n3. Marathi\n4. Telugu",
			"returning_menu":          "Menu:\n1. Application Status\n2. Profile Update\n3. Jobs Browse\n4. Job Apply",
			"status_otp_sent":         "6 digit OTP pathavla aahe. Status pahanyasathi OTP reply kara.",
			"invalid_otp":             "OTP chukicha aahe. Punha try kara.",
			"ask_target_role":         "Tumhi konta job role shodhat aahat?",
			"ask_preferred_zone":      "Tumhala konti location kiwa zone pahije?",
			"ask_verification_option": "Verification option nivda:\nA. Aadhaar OTP\nB. Document Upload\nC. Skip Verification",
			"invalid_menu_option":     "Invalid menu option.",
		},
		"te": {
			"new_user_greeting":       "BlueCollarJob ki swagatham. Mee full name cheppandi.",
			"ask_language":            "Language select cheyyandi:\n1. English\n2. Hindi\n3. Marathi\n4. Telugu",
			"returning_menu":          "Menu:\n1. Application Status\n2. Profile Update\n3. Jobs Browse\n4. Job Apply",
			"status_otp_sent":         "6 digit OTP pampincham. Status kosam OTP reply cheyyandi.",
			"invalid_otp":             "OTP tappu. Malli try cheyyandi.",
			"ask_target_role":         "Meeru ye job role kavali?",
			"ask_preferred_zone":      "Mee preferred location or zone enti?",
			"ask_verification_option": "Verification option select cheyyandi:\nA. Aadhaar OTP\nB. Document Upload\nC. Skip Verification",
			"invalid_menu_option":     "Invalid menu option.",
		},
	}
	if reply, ok := replies[language][key]; ok {
		return reply
	}
	if reply, ok := replies["en"][key]; ok {
		return reply
	}
	return "Thanks. We will continue shortly."
}

func buildJobsReply(language string, jobs []models.Job) string {
	if len(jobs) == 0 {
		return localizedReply(language, "no_jobs")
	}
	lines := []string{"Active jobs:"}
	for _, job := range jobs {
		lines = append(lines, fmt.Sprintf("- %s | %s | %s | %s, %s | %s", job.ID, job.Title, job.Role, job.LocationCity, job.LocationState, job.ShiftSchedule))
	}
	return strings.Join(lines, "\n")
}

func isYes(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "yes", "y", "haan", "ha", "ok":
		return true
	default:
		return false
	}
}

func isReferralCommand(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return value == "referral" || value == "referrals" || value == "my referral"
}

func parseReferralCodeCommand(value string) string {
	parts := strings.Fields(strings.TrimSpace(value))
	if len(parts) == 2 && (strings.EqualFold(parts[0], "ref") || strings.EqualFold(parts[0], "referral")) {
		return strings.TrimSpace(parts[1])
	}
	return ""
}

func buildApplicationStatusReply(language string, applications []models.Application) string {
	if len(applications) == 0 {
		return localizedReply(language, "no_applications")
	}
	lines := []string{"Your applications:"}
	for _, application := range applications {
		lines = append(lines, fmt.Sprintf("- %s: %s", application.ID, application.Status))
	}
	return strings.Join(lines, "\n")
}

func referralCodeFromPhone(phoneNumber string) string {
	digits := onlyDigits(phoneNumber)
	return "WA" + lastDigits(digits, 10)
}

func lastDigits(value string, count int) string {
	if len(value) <= count {
		return value
	}
	return value[len(value)-count:]
}

func onlyDigits(value string) string {
	var builder strings.Builder
	for _, r := range value {
		if r >= '0' && r <= '9' {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

func fallbackName(user *models.User, data botStateData) string {
	if strings.TrimSpace(data.FullName) != "" {
		return strings.TrimSpace(data.FullName)
	}
	return user.FullName
}

func stringPtrFromValue(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}
