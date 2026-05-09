package persistence

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/rent-a-girlfriend/identity-service/internal/domain/aggregate"
	domainerr "github.com/rent-a-girlfriend/identity-service/internal/domain/errors"
	"github.com/rent-a-girlfriend/identity-service/internal/domain/vo"
)

// UserAccountRepoImpl implements UserAccountRepository using GORM.
type UserAccountRepoImpl struct {
	db *gorm.DB
}

// NewUserAccountRepoImpl creates a new repository implementation.
func NewUserAccountRepoImpl(db *gorm.DB) *UserAccountRepoImpl {
	return &UserAccountRepoImpl{db: db}
}

func (r *UserAccountRepoImpl) Save(ctx context.Context, account *aggregate.UserAccount) error {
	model := toUserAccountModel(account)
	return r.db.WithContext(ctx).Create(model).Error
}

func (r *UserAccountRepoImpl) Update(ctx context.Context, account *aggregate.UserAccount) error {
	model := toUserAccountModel(account)
	result := r.db.WithContext(ctx).
		Model(&UserAccountModel{}).
		Where("id = ? AND version = ?", model.ID, model.Version-1).
		Updates(model)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domainerr.ErrConcurrencyConflict
	}
	return nil
}

func (r *UserAccountRepoImpl) FindByID(ctx context.Context, id vo.UserID) (*aggregate.UserAccount, error) {
	var model UserAccountModel
	if err := r.db.WithContext(ctx).Where("id = ?", id.UUID()).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domainerr.ErrAccountNotFound
		}
		return nil, fmt.Errorf("failed to find account by ID: %w", err)
	}
	return toUserAccountAggregate(&model)
}

func (r *UserAccountRepoImpl) FindByEmail(ctx context.Context, email vo.Email) (*aggregate.UserAccount, error) {
	var model UserAccountModel
	if err := r.db.WithContext(ctx).Where("email = ?", email.String()).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domainerr.ErrAccountNotFound
		}
		return nil, fmt.Errorf("failed to find account by email: %w", err)
	}
	return toUserAccountAggregate(&model)
}

func (r *UserAccountRepoImpl) FindByGoogleID(ctx context.Context, googleID string) (*aggregate.UserAccount, error) {
	var model UserAccountModel
	if err := r.db.WithContext(ctx).Where("google_id = ?", googleID).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domainerr.ErrAccountNotFound
		}
		return nil, fmt.Errorf("failed to find account by google ID: %w", err)
	}
	return toUserAccountAggregate(&model)
}

// --- Mapping helpers ---

func toUserAccountModel(a *aggregate.UserAccount) *UserAccountModel {
	return &UserAccountModel{
		ID:             a.ID().UUID(),
		Email:          a.Email().String(),
		GoogleID:       a.GoogleID(),
		Role:           string(a.Role()),
		Status:         string(a.Status()),
		ViolationCount: a.ViolationCount(),
		Version:        a.Version() + 1, // increment for optimistic locking
		CreatedAt:      a.CreatedAt(),
		UpdatedAt:      a.UpdatedAt(),
	}
}

func toUserAccountAggregate(m *UserAccountModel) (*aggregate.UserAccount, error) {
	id, err := vo.ParseUserID(m.ID.String())
	if err != nil {
		return nil, err
	}
	email, err := vo.NewEmail(m.Email)
	if err != nil {
		return nil, err
	}

	return aggregate.Reconstitute(
		id,
		email,
		m.GoogleID,
		vo.Role(m.Role),
		vo.AccountStatus(m.Status),
		m.ViolationCount,
		m.Version,
		m.CreatedAt,
		m.UpdatedAt,
	), nil
}
