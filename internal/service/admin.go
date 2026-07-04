package service

import (
	"context"
	"fmt"
	"strings"

	"bluecollarjob/internal/models"
	"bluecollarjob/internal/repository"
)

type AdminService interface {
	GetSummary(ctx context.Context) (*models.AdminSummary, error)
	ListReferralTransactions(ctx context.Context, filters AdminReferralTransactionFilters) ([]models.ReferralTransaction, error)
}

type AdminReferralTransactionFilters struct {
	Status string
	Limit  int
	Offset int
}

type adminService struct {
	admin     repository.AdminRepository
	referrals repository.ReferralRepository
}

func NewAdminService(admin repository.AdminRepository, referrals repository.ReferralRepository) AdminService {
	return &adminService{admin: admin, referrals: referrals}
}

func (s *adminService) GetSummary(ctx context.Context) (*models.AdminSummary, error) {
	return s.admin.GetSummary(ctx)
}

func (s *adminService) ListReferralTransactions(ctx context.Context, filters AdminReferralTransactionFilters) ([]models.ReferralTransaction, error) {
	status := strings.TrimSpace(filters.Status)
	repoFilters := repository.ReferralTransactionFilters{
		Limit:  filters.Limit,
		Offset: filters.Offset,
	}
	if status != "" {
		if !validReferralTransactionStatus(status) {
			return nil, fmt.Errorf("%w: invalid referral transaction status", ErrInvalidInput)
		}
		repoFilters.Status = &status
	}
	return s.referrals.ListReferralTransactions(ctx, repoFilters)
}

func validReferralTransactionStatus(status string) bool {
	switch status {
	case "Pending", "Processing", "Paid", "Failed":
		return true
	default:
		return false
	}
}
