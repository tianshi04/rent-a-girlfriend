package persistence

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/rent-a-girlfriend/booking-service/internal/domain/aggregate"
	domainerr "github.com/rent-a-girlfriend/booking-service/internal/domain/errors"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/repository"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/vo"
)

// BookingRepositoryImpl is the GORM implementation of BookingRepository.
type BookingRepositoryImpl struct {
	db *gorm.DB
}

func NewBookingRepository(db *gorm.DB) *BookingRepositoryImpl {
	return &BookingRepositoryImpl{db: db}
}

func (r *BookingRepositoryImpl) Save(ctx context.Context, booking *aggregate.Booking) error {
	model := ToModel(booking)
	result := r.db.WithContext(ctx).Create(model)
	return result.Error
}

func (r *BookingRepositoryImpl) Update(ctx context.Context, booking *aggregate.Booking) error {
	model := ToModel(booking)
	oldVersion := model.Version
	model.Version = oldVersion + 1

	// Optimistic locking: only update if version matches
	result := r.db.WithContext(ctx).
		Model(&BookingModel{}).
		Where("id = ? AND version = ?", model.ID, oldVersion).
		Clauses(clause.Returning{}).
		Updates(map[string]interface{}{
			"status":           model.Status,
			"cancelled_by_role": model.CancelledByRole,
			"is_late_cancel":   model.IsLateCancel,
			"version":          model.Version,
			"updated_at":       model.UpdatedAt,
		})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domainerr.ErrConcurrencyConflict
	}
	return nil
}

func (r *BookingRepositoryImpl) FindByID(ctx context.Context, id vo.BookingID) (*aggregate.Booking, error) {
	var model BookingModel
	result := r.db.WithContext(ctx).Where("id = ?", id.String()).First(&model)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, domainerr.ErrBookingNotFound
		}
		return nil, result.Error
	}
	return model.ToDomain()
}

func (r *BookingRepositoryImpl) CountPendingByClientAndCompanion(
	ctx context.Context,
	clientID vo.ClientID,
	companionID vo.CompanionID,
) (int, error) {
	var count int64
	result := r.db.WithContext(ctx).
		Model(&BookingModel{}).
		Where("client_id = ? AND companion_id = ? AND status = ?",
			clientID.String(), companionID.String(), string(vo.StatusPending)).
		Count(&count)
	if result.Error != nil {
		return 0, result.Error
	}
	return int(count), nil
}

func (r *BookingRepositoryImpl) FindByFilters(
	ctx context.Context,
	filters repository.BookingFilters,
) ([]*aggregate.Booking, int64, error) {
	query := r.db.WithContext(ctx).Model(&BookingModel{})

	if filters.ClientID != nil {
		query = query.Where("client_id = ?", *filters.ClientID)
	}
	if filters.CompanionID != nil {
		query = query.Where("companion_id = ?", *filters.CompanionID)
	}
	if filters.Status != nil {
		query = query.Where("status = ?", *filters.Status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (filters.Page - 1) * filters.PageSize
	var models []BookingModel
	if err := query.Order("created_at DESC").Offset(offset).Limit(filters.PageSize).Find(&models).Error; err != nil {
		return nil, 0, err
	}

	bookings := make([]*aggregate.Booking, 0, len(models))
	for i := range models {
		b, err := models[i].ToDomain()
		if err != nil {
			return nil, 0, err
		}
		bookings = append(bookings, b)
	}

	return bookings, total, nil
}
