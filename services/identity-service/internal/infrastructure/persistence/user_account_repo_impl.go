package persistence

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/rent-a-girlfriend/identity-service/internal/domain/aggregate"
	domainerr "github.com/rent-a-girlfriend/identity-service/internal/domain/errors"
	"github.com/rent-a-girlfriend/identity-service/internal/domain/vo"
	"github.com/rent-a-girlfriend/identity-service/internal/infrastructure/cache"
)

// UserAccountRepoImpl implements UserAccountRepository using GORM.
type UserAccountRepoImpl struct {
	db    *gorm.DB
	cache *cache.RedisAdapter
}

// NewUserAccountRepoImpl creates a new repository implementation.
func NewUserAccountRepoImpl(db *gorm.DB, cache *cache.RedisAdapter) *UserAccountRepoImpl {
	return &UserAccountRepoImpl{
		db:    db,
		cache: cache,
	}
}

func (r *UserAccountRepoImpl) getIDKey(id string) string {
	return fmt.Sprintf("account:id:%s", id)
}

func (r *UserAccountRepoImpl) getEmailKey(email string) string {
	return fmt.Sprintf("account:email:%s", email)
}

func (r *UserAccountRepoImpl) Save(ctx context.Context, account *aggregate.UserAccount) error {
	model := toUserAccountModel(account)
	err := r.db.WithContext(ctx).Create(model).Error
	if err != nil {
		return err
	}

	// Evict cache
	_ = r.cache.Delete(ctx, r.getIDKey(model.ID.String()))
	_ = r.cache.Delete(ctx, r.getEmailKey(model.Email))
	_ = r.cache.Delete(ctx, r.getGoogleIDKey(model.GoogleID))
	return nil
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

	// Evict cache
	_ = r.cache.Delete(ctx, r.getIDKey(model.ID.String()))
	_ = r.cache.Delete(ctx, r.getEmailKey(model.Email))
	_ = r.cache.Delete(ctx, r.getGoogleIDKey(model.GoogleID))
	return nil
}

func (r *UserAccountRepoImpl) FindByID(ctx context.Context, id vo.UserID) (*aggregate.UserAccount, error) {
	key := r.getIDKey(id.String())
	var model UserAccountModel

	// Try cache
	found, _ := r.cache.Get(ctx, key, &model)
	if found {
		return toUserAccountAggregate(&model)
	}

	// Cache miss, try DB
	if err := r.db.WithContext(ctx).Where("id = ?", id.UUID()).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domainerr.ErrAccountNotFound
		}
		return nil, fmt.Errorf("failed to find account by ID: %w", err)
	}

	// Save to cache for 1 hour
	_ = r.cache.Set(ctx, key, &model, 1*time.Hour)
	return toUserAccountAggregate(&model)
}

func (r *UserAccountRepoImpl) FindByEmail(ctx context.Context, email vo.Email) (*aggregate.UserAccount, error) {
	key := r.getEmailKey(email.String())
	var model UserAccountModel

	// Try cache
	found, _ := r.cache.Get(ctx, key, &model)
	if found {
		return toUserAccountAggregate(&model)
	}

	// Cache miss, try DB
	if err := r.db.WithContext(ctx).Where("email = ?", email.String()).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domainerr.ErrAccountNotFound
		}
		return nil, fmt.Errorf("failed to find account by email: %w", err)
	}

	// Save to cache
	_ = r.cache.Set(ctx, key, &model, 1*time.Hour)
	return toUserAccountAggregate(&model)
}

func (r *UserAccountRepoImpl) getGoogleIDKey(googleID string) string {
	return fmt.Sprintf("account:google_id:%s", googleID)
}

func (r *UserAccountRepoImpl) FindByGoogleID(ctx context.Context, googleID string) (*aggregate.UserAccount, error) {
	key := r.getGoogleIDKey(googleID)
	var model UserAccountModel

	// Try cache
	found, _ := r.cache.Get(ctx, key, &model)
	if found {
		return toUserAccountAggregate(&model)
	}

	// Cache miss, try DB
	if err := r.db.WithContext(ctx).Where("google_id = ?", googleID).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domainerr.ErrAccountNotFound
		}
		return nil, fmt.Errorf("failed to find account by google ID: %w", err)
	}

	// Save to cache for 1 hour
	_ = r.cache.Set(ctx, key, &model, 1*time.Hour)
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
