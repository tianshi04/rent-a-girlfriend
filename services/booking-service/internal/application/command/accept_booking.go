package command

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	financev1 "github.com/rent-a-girlfriend/booking-service/gen/proto/financev1"
	"github.com/rent-a-girlfriend/booking-service/internal/application/port"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/aggregate"
	domainerr "github.com/rent-a-girlfriend/booking-service/internal/domain/errors"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/event"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/repository"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/vo"
)

// AcceptBookingCmd holds the data for accepting a booking.
type AcceptBookingCmd struct {
	BookingID   string
	CompanionID string
}

// AcceptBookingHandler initiates the async Accept SAGA.
// It only: (1) creates a BookingAcceptSaga in WAITING_FOR_ESCROW, and
// (2) writes a TransferToEscrowCommand into the outbox — then returns.
// All subsequent SAGA steps are driven by the SagaCoordinator reacting to Kafka events.
type AcceptBookingHandler struct {
	bookingRepo repository.BookingRepository
	sagaRepo    repository.BookingSagaRepository
	db          *gorm.DB
	outbox      port.EventPublisher
}

func NewAcceptBookingHandler(
	bookingRepo repository.BookingRepository,
	sagaRepo repository.BookingSagaRepository,
	db *gorm.DB,
	outbox port.EventPublisher,
) *AcceptBookingHandler {
	return &AcceptBookingHandler{
		bookingRepo: bookingRepo,
		sagaRepo:    sagaRepo,
		db:          db,
		outbox:      outbox,
	}
}

func (h *AcceptBookingHandler) Handle(ctx context.Context, cmd AcceptBookingCmd) (*aggregate.Booking, error) {
	bookingID, err := vo.BookingIDFromString(cmd.BookingID)
	if err != nil {
		return nil, err
	}
	companionID, err := vo.NewCompanionID(cmd.CompanionID)
	if err != nil {
		return nil, err
	}

	booking, err := h.bookingRepo.FindByID(ctx, bookingID)
	if err != nil {
		return nil, err
	}

	// Validate companion owns the booking before kicking off SAGA
	if !booking.CompanionID().Equals(companionID) {
		return nil, domainerr.ErrUnauthorized
	}

	// Check if companion has any other overlapping booking in ACCEPTED state
	companionOverlap, err := h.bookingRepo.HasOverlappingBooking(
		ctx,
		booking.CompanionID().String(),
		true,
		[]vo.BookingStatus{vo.StatusAccepted},
		booking.TimeRange().StartTime(),
		booking.TimeRange().EndTime(),
	)
	if err != nil {
		return nil, err
	}
	if companionOverlap {
		return nil, domainerr.ErrCompanionBookingOverlap
	}

	// --- Single DB transaction: create Saga + write outbox command ---
	sagaID := uuid.New().String()
	saga := aggregate.NewBookingAcceptSaga(sagaID, bookingID.String(), time.Now())
	saga.UpdateState(vo.SagaStateWaitingForEscrow, time.Now())

	cmd2 := event.TransferToEscrowCommand{
		TransferToEscrowRequest: &financev1.TransferToEscrowRequest{
			BookingId: booking.ID().String(),
			UserId:    booking.ClientID().String(),
			Amount:    booking.Scenario().Price().Amount(),
		},
		Timestamp: time.Now(),
	}

	if h.db != nil {
		// Production path: wrap saga save + outbox publish in a single transaction
		tx := h.db.WithContext(ctx).Begin()
		if tx.Error != nil {
			return nil, tx.Error
		}
		txCtx := context.WithValue(ctx, vo.TxKey, tx)

		if err := h.sagaRepo.Save(txCtx, saga); err != nil {
			tx.Rollback()
			return nil, err
		}
		if err := h.outbox.Publish(txCtx, cmd2); err != nil {
			tx.Rollback()
			return nil, err
		}
		if err := tx.Commit().Error; err != nil {
			return nil, err
		}
	} else {
		// Test path: no DB transaction (outbox is a no-op mock)
		if err := h.sagaRepo.Save(ctx, saga); err != nil {
			return nil, err
		}
		if err := h.outbox.Publish(ctx, cmd2); err != nil {
			return nil, err
		}
	}

	// Return booking immediately — SAGA completes asynchronously
	return booking, nil
}
