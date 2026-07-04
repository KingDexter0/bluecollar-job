package handler

import (
	"net/http"

	"bluecollarjob/internal/service"

	"github.com/gin-gonic/gin"
)

type ReferralHandler struct {
	referrals service.ReferralService
}

func NewReferralHandler(referrals service.ReferralService) *ReferralHandler {
	return &ReferralHandler{referrals: referrals}
}

func (h *ReferralHandler) GetReferral(c *gin.Context) {
	worker, err := h.referrals.GetWorkerReferral(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"worker_id":     worker.ID,
		"referral_code": worker.ReferralCode,
	})
}

func (h *ReferralHandler) ListReferrals(c *gin.Context) {
	limit, offset := parsePagination(c)
	referrals, err := h.referrals.ListReferrals(c.Request.Context(), c.Param("id"), limit, offset)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response := make([]referralResponse, 0, len(referrals))
	for _, referral := range referrals {
		response = append(response, newReferralResponse(referral))
	}
	c.JSON(http.StatusOK, gin.H{"referrals": response})
}

func (h *ReferralHandler) ListTransactions(c *gin.Context) {
	limit, offset := parsePagination(c)
	transactions, err := h.referrals.ListTransactions(c.Request.Context(), c.Param("id"), limit, offset)
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
