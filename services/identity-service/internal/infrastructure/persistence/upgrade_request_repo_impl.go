package persistence

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/rent-a-girlfriend/identity-service/internal/domain/aggregate"
	domainerr "github.com/rent-a-girlfriend/identity-service/internal/domain/errors"
	"github.com/rent-a-girlfriend/identity-service/internal/domain/repository"
	"github.com/rent-a-girlfriend/identity-service/internal/domain/vo"
)

// UpgradeRequestRepoImpl implements UpgradeRequestRepository using GORM.
type UpgradeRequestRepoImpl struct {
	db *gorm.DB
}

// NewUpgradeRequestRepoImpl creates a new repository implementation.
func NewUpgradeRequestRepoImpl(db *gorm.DB) *UpgradeRequestRepoImpl {
	return &UpgradeRequestRepoImpl{db: db}
}

func (r *UpgradeRequestRepoImpl) Save(ctx context.Context, req *aggregate.UpgradeRequest) error {
	model := toUpgradeRequestModel(req)
	return r.db.WithContext(ctx).Create(model).Error
}

func (r *UpgradeRequestRepoImpl) Update(ctx context.Context, req *aggregate.UpgradeRequest) error {
	model := toUpgradeRequestModel(req)
	return r.db.WithContext(ctx).Save(model).Error
}

func (r *UpgradeRequestRepoImpl) FindByID(ctx context.Context, id uuid.UUID) (*aggregate.UpgradeRequest, error) {
	var model UpgradeRequestModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domainerr.ErrUpgradeRequestNotFound
		}
		return nil, fmt.Errorf("failed to find upgrade request: %w", err)
	}
	return toUpgradeRequestAggregate(&model)
}

func (r *UpgradeRequestRepoImpl) FindPendingByUserID(ctx context.Context, userID vo.UserID) (*aggregate.UpgradeRequest, error) {
	var model UpgradeRequestModel
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND status = ?", userID.UUID(), "PENDING").
		First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domainerr.ErrUpgradeRequestNotFound
		}
		return nil, fmt.Errorf("failed to find pending upgrade request: %w", err)
	}
	return toUpgradeRequestAggregate(&model)
}

func (r *UpgradeRequestRepoImpl) FindByFilters(ctx context.Context, filters repository.UpgradeRequestFilters) ([]*aggregate.UpgradeRequest, int64, error) {
	query := r.db.WithContext(ctx).Model(&UpgradeRequestModel{})

	if filters.Status != nil {
		query = query.Where("status = ?", *filters.Status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (filters.Page - 1) * filters.PageSize
	var models []UpgradeRequestModel
	if err := query.Order("created_at DESC").Offset(offset).Limit(filters.PageSize).Find(&models).Error; err != nil {
		return nil, 0, err
	}

	requests := make([]*aggregate.UpgradeRequest, 0, len(models))
	for _, m := range models {
		req, err := toUpgradeRequestAggregate(&m)
		if err != nil {
			return nil, 0, err
		}
		requests = append(requests, req)
	}

	return requests, total, nil
}

func toUpgradeRequestModel(r *aggregate.UpgradeRequest) *UpgradeRequestModel {
	return &UpgradeRequestModel{
		ID:           r.ID(),
		UserID:       r.UserID().UUID(),
		Status:       string(r.Status()),
		Reason:       r.Reason(),
		RejectReason: r.RejectReason(),
		ReviewedBy:   r.ReviewedBy(),
		ReviewedAt:   r.ReviewedAt(),
		CreatedAt:    r.CreatedAt(),
	}
}

func toUpgradeRequestAggregate(m *UpgradeRequestModel) (*aggregate.UpgradeRequest, error) {
	userID, err := vo.ParseUserID(m.UserID.String())
	if err != nil {
		return nil, err
	}

	return aggregate.ReconstituteUpgradeRequest(
		m.ID,
		userID,
		vo.UpgradeStatus(m.Status),
		m.Reason,
		m.RejectReason,
		m.ReviewedBy,
		m.ReviewedAt,
		m.CreatedAt,
	), nil
}
