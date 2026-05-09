package service

import (
	"context"

	"github.com/rent-a-girlfriend/identity-service/internal/domain/repository"
)

const (
	// ConfigKeyViolationLockThreshold is the DB config key for the auto-lock threshold.
	ConfigKeyViolationLockThreshold = "violation_lock_threshold"

	// DefaultViolationLockThreshold is the default threshold if not configured.
	DefaultViolationLockThreshold = 3
)

// AccountLockPolicyService determines whether an account should be locked
// based on its violation count and the configurable threshold from DB.
type AccountLockPolicyService struct {
	configRepo repository.SystemConfigRepository
}

// NewAccountLockPolicyService creates a new policy service.
func NewAccountLockPolicyService(configRepo repository.SystemConfigRepository) *AccountLockPolicyService {
	return &AccountLockPolicyService{configRepo: configRepo}
}

// ShouldLock checks if the given violation count has reached or exceeded the threshold.
func (s *AccountLockPolicyService) ShouldLock(ctx context.Context, violationCount int) (bool, error) {
	threshold, err := s.configRepo.GetInt(ctx, ConfigKeyViolationLockThreshold, DefaultViolationLockThreshold)
	if err != nil {
		return false, err
	}
	return violationCount >= threshold, nil
}
