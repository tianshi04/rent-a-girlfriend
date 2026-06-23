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

// RejectBookingCmd holds the data for rejecting a booking.
type RejectBookingCmd struct {
	BookingID   string
	CompanionID string
	Reason      string
}

// RejectBookingHandler handles the RejectBooking command.
type RejectBookingHandler struct {
	repo   repository.BookingRepository
	db     *gorm.DB
	outbox port.EventPublisher
}

func NewRejectBookingHandler(repo repository.BookingRepository, db *gorm.DB, outbox port.EventPublisher) *RejectBookingHandler {
	return &RejectBookingHandler{repo: repo, db: db, outbox: outbox}
}

func (h *RejectBookingHandler) Handle(ctx context.Context, cmd RejectBookingCmd) (*aggregate.Booking, error) {
	bookingID, err := vo.BookingIDFromString(cmd.BookingID)
	if err != nil {
		return nil, err
	}
	companionID, err := vo.NewCompanionID(cmd.CompanionID)
	if err != nil {
		return nil, err
	}

	booking, err := h.repo.FindByID(ctx, bookingID)
	if err != nil {
		return nil, err
	}

	if err := booking.Reject(companionID, cmd.Reason, time.Now()); err != nil {
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
