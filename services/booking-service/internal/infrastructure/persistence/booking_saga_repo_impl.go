package persistence

import (
	"context"

	"gorm.io/gorm"

	"github.com/rent-a-girlfriend/booking-service/internal/domain/aggregate"
)

type BookingSagaRepoImpl struct {
	db *gorm.DB
}

func NewBookingSagaRepo(db *gorm.DB) *BookingSagaRepoImpl {
	return &BookingSagaRepoImpl{db: db}
}

func (r *BookingSagaRepoImpl) txDB(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value("tx").(*gorm.DB); ok {
		return tx.WithContext(ctx)
	}
	return r.db.WithContext(ctx)
}

func (r *BookingSagaRepoImpl) Save(ctx context.Context, saga *aggregate.BookingAcceptSaga) error {
	model := ToSagaModel(saga)
	return r.txDB(ctx).Create(model).Error
}

func (r *BookingSagaRepoImpl) Update(ctx context.Context, saga *aggregate.BookingAcceptSaga) error {
	model := ToSagaModel(saga)
	return r.txDB(ctx).Save(model).Error
}

func (r *BookingSagaRepoImpl) FindByID(ctx context.Context, id string) (*aggregate.BookingAcceptSaga, error) {
	var model BookingAcceptSagaModel
	err := r.txDB(ctx).Where("id = ?", id).First(&model).Error
	if err != nil {
		return nil, err
	}
	return model.ToDomain(), nil
}

func (r *BookingSagaRepoImpl) FindByBookingID(ctx context.Context, bookingID string) (*aggregate.BookingAcceptSaga, error) {
	var model BookingAcceptSagaModel
	err := r.txDB(ctx).Where("booking_id = ?", bookingID).Order("created_at desc").First(&model).Error
	if err != nil {
		return nil, err
	}
	return model.ToDomain(), nil
}
