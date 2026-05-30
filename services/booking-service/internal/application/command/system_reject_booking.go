package command

import (
	"context"
	"time"

	"github.com/rent-a-girlfriend/booking-service/internal/application/port"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/aggregate"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/repository"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/vo"
)

type SystemRejectBookingCmd struct {
	BookingID string
}

type SystemRejectBookingHandler struct {
	repo           repository.BookingRepository
	financeService port.FinanceService
}

func NewSystemRejectBookingHandler(repo repository.BookingRepository, financeService port.FinanceService) *SystemRejectBookingHandler {
	return &SystemRejectBookingHandler{
		repo:           repo,
		financeService: financeService,
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

	// Unfreeze coin since booking is rejected due to timeout
	if err := h.financeService.UnfreezeCoin(ctx, booking.ClientID(), booking.Scenario().Price()); err != nil {
		// Log error but don't fail the reject
	}

	if err := h.repo.Update(ctx, booking); err != nil {
		return nil, err
	}

	return booking, nil
}
