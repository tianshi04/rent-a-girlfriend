package command

import (
	"context"
	"time"

	"gorm.io/gorm"

	financev1 "github.com/rent-a-girlfriend/booking-service/gen/proto/financev1"
	"github.com/rent-a-girlfriend/booking-service/internal/application/port"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/aggregate"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/event"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/repository"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/vo"
)

type SystemRejectBookingCmd struct {
	BookingID string
}

type SystemRejectBookingHandler struct {
	repo   repository.BookingRepository
	db     *gorm.DB
	outbox port.EventPublisher
}

func NewSystemRejectBookingHandler(repo repository.BookingRepository, db *gorm.DB, outbox port.EventPublisher) *SystemRejectBookingHandler {
	return &SystemRejectBookingHandler{
		repo:   repo,
		db:     db,
		outbox: outbox,
	}
}

func (h *SystemRejectBookingHandler) Handle(ctx context.Context, cmd SystemRejectBookingCmd) (*aggregate.Booking, error) {
	bookingID, err := vo.BookingIDFromString(cmd.BookingID)
	if err != nil {
		return nil, err
	}

	booking, err := h.repo.FindByID(ctx, bookingID)
	if err != nil {
		return nil, err
	}

	if err := booking.SystemTimeout(time.Now()); err != nil {
		return nil, err
	}

	err = h.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txCtx := context.WithValue(ctx, vo.TxKey, tx)

		if err := h.repo.Update(txCtx, booking); err != nil {
			return err
		}

		return h.outbox.Publish(txCtx, event.UnfreezeCoinCommand{
			UnfreezeCoin: &financev1.UnfreezeCoin{
				UserId:    booking.ClientID().String(),
				Amount:    booking.Scenario().Price().Amount(),
				BookingId: booking.ID().String(),
			},
			Timestamp: time.Now(),
		})
	})
	if err != nil {
		return nil, err
	}

	return booking, nil
}
