package handler

import (
	"net/http"

	"bluecollarjob/internal/models"
	"bluecollarjob/internal/service"

	"github.com/gin-gonic/gin"
)

type AdminHandler struct {
	admin         service.AdminService
	notifications service.NotificationQueryService
	referrals     service.ReferralService
}

func NewAdminHandler(admin service.AdminService, notifications service.NotificationQueryService, referrals service.ReferralService) *AdminHandler {
	return &AdminHandler{admin: admin, notifications: notifications, referrals: referrals}
}

func (h *AdminHandler) GetSummary(c *gin.Context) {
	summary, err := h.admin.GetSummary(c.Request.Context())
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"summary": newAdminSummaryResponse(summary)})
}

func (h *AdminHandler) ListNotifications(c *gin.Context) {
	limit, offset := parsePagination(c)
	notifications, err := h.notifications.ListNotifications(c.Request.Context(), service.NotificationListFilters{
		Status:    c.Query("status"),
		EventType: c.Query("event_type"),
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"notifications": notifications})
}

func (h *AdminHandler) ListReferralTransactions(c *gin.Context) {
	limit, offset := parsePagination(c)
	transactions, err := h.admin.ListReferralTransactions(c.Request.Context(), service.AdminReferralTransactionFilters{
		Status: c.Query("status"),
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response := make([]referralTransactionResponse, 0, len(transactions))
	for _, transaction := range transactions {
		response = append(response, newReferralTransactionResponse(transaction))
	}
	c.JSON(http.StatusOK, gin.H{"referral_transactions": response})
}

func (h *AdminHandler) ProcessReferralPayouts(c *gin.Context) {
	var request devProcessReferralPayoutsRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		request.Limit = 10
	}
	if request.Limit <= 0 {
		request.Limit = 10
	}
	result, err := h.referrals.ProcessPendingPayouts(c.Request.Context(), request.Limit)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"result": result})
}

type adminSummaryResponse struct {
	TotalWorkers              int64                              `json:"total_workers"`
	TotalEmployers            int64                              `json:"total_employers"`
	TotalJobs                 int64                              `json:"total_jobs"`
	TotalApplications         int64                              `json:"total_applications"`
	TotalReferrals            int64                              `json:"total_referrals"`
	TotalNotificationEvents   int64                              `json:"total_notification_events"`
	PendingNotifications      int64                              `json:"pending_notifications"`
	FailedNotifications       int64                              `json:"failed_notifications"`
	CashbackPending           int64                              `json:"cashback_pending"`
	CashbackPaid              int64                              `json:"cashback_paid"`
	CashbackFailed            int64                              `json:"cashback_failed"`
	InterviewsScheduled       int64                              `json:"interviews_scheduled"`
	ApplicationsByStatus      map[models.ApplicationStatus]int64 `json:"applications_by_status"`
	WorkersByVerificationTier map[models.VerificationTier]int64  `json:"workers_by_verification_tier"`
	JobsByActiveStatus        map[string]int64                   `json:"jobs_by_active_status"`
	ReferralsByPayoutStatus   map[string]int64                   `json:"referrals_by_payout_status"`
}

func newAdminSummaryResponse(summary *models.AdminSummary) adminSummaryResponse {
	return adminSummaryResponse{
		TotalWorkers:              summary.TotalWorkers,
		TotalEmployers:            summary.TotalEmployers,
		TotalJobs:                 summary.TotalJobs,
		TotalApplications:         summary.TotalApplications,
		TotalReferrals:            summary.TotalReferrals,
		TotalNotificationEvents:   summary.TotalNotificationEvents,
		PendingNotifications:      summary.PendingNotifications,
		FailedNotifications:       summary.FailedNotifications,
		CashbackPending:           summary.CashbackPending,
		CashbackPaid:              summary.CashbackPaid,
		CashbackFailed:            summary.CashbackFailed,
		InterviewsScheduled:       summary.InterviewsScheduled,
		ApplicationsByStatus:      summary.ApplicationsByStatus,
		WorkersByVerificationTier: summary.WorkersByVerificationTier,
		JobsByActiveStatus:        summary.JobsByActiveStatus,
		ReferralsByPayoutStatus:   summary.ReferralsByPayoutStatus,
	}
}
