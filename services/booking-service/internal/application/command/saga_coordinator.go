package command

import (
	"context"
	"log"
	"time"

	"gorm.io/gorm"

	"google.golang.org/protobuf/types/known/timestamppb"

	financev1 "github.com/rent-a-girlfriend/booking-service/gen/proto/financev1"
	financev1events "github.com/rent-a-girlfriend/booking-service/gen/proto/financev1/events"
	interactionv1 "github.com/rent-a-girlfriend/booking-service/gen/proto/interactionv1"
	"github.com/rent-a-girlfriend/booking-service/internal/application/port"
	domainerr "github.com/rent-a-girlfriend/booking-service/internal/domain/errors"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/event"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/repository"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/vo"
	"github.com/rent-a-girlfriend/booking-service/internal/infrastructure/persistence"
)

// SagaCoordinator handles inbound async events from Kafka and drives the
// BookingAcceptSaga through its state machine.
type SagaCoordinator struct {
	bookingRepo repository.BookingRepository
	sagaRepo    repository.BookingSagaRepository
	db          *gorm.DB
	outbox      port.EventPublisher
}

func NewSagaCoordinator(
	bookingRepo repository.BookingRepository,
	sagaRepo repository.BookingSagaRepository,
	db *gorm.DB,
	outbox port.EventPublisher,
) *SagaCoordinator {
	return &SagaCoordinator{
		bookingRepo: bookingRepo,
		sagaRepo:    sagaRepo,
		db:          db,
		outbox:      outbox,
	}
}

// HandleEscrowSuccess is called when Finance emits CoinEscrowed for a booking.
// Saga: WAITING_FOR_ESCROW -> WAITING_FOR_CHAT. Writes CreateChatRoomCommand to outbox.
func (c *SagaCoordinator) HandleEscrowSuccess(ctx context.Context, bookingID string, eventID string) error {
	return c.withTx(ctx, func(txCtx context.Context) error {
		alreadyProcessed, err := persistence.CheckAndRecordEvent(txCtx, c.db, eventID, "finance.escrow-created.v1")
		if err != nil {
			return err
		}
		if alreadyProcessed {
			return nil
		}

		booking, err := c.bookingRepo.FindByID(txCtx, mustBookingID(bookingID))
		if err != nil {
			return err
		}
		saga, err := c.sagaRepo.FindByBookingID(txCtx, bookingID)
		if err != nil {
			return err
		}

		saga.UpdateState(vo.SagaStateWaitingForChat, time.Now())
		if err := c.sagaRepo.Update(txCtx, saga); err != nil {
			return err
		}

		return c.outbox.Publish(txCtx, event.CreateChatRoomCommand{
			CreateChatRoomRequest: &interactionv1.CreateChatRoomRequest{
				BookingId:   bookingID,
				ClientId:    booking.ClientID().String(),
				CompanionId: booking.CompanionID().String(),
			},
			Timestamp: time.Now(),
		})
	})
}

// HandleEscrowFailed is called when Finance emits EscrowFailed for a booking.
// Saga: WAITING_FOR_ESCROW -> FAILED_TECHNICAL. Booking -> CANCELLED.
func (c *SagaCoordinator) HandleEscrowFailed(ctx context.Context, bookingID string, eventID string) error {
	return c.withTx(ctx, func(txCtx context.Context) error {
		alreadyProcessed, err := persistence.CheckAndRecordEvent(txCtx, c.db, eventID, "finance.escrow-failed.v1")
		if err != nil {
			return err
		}
		if alreadyProcessed {
			return nil
		}

		booking, err := c.bookingRepo.FindByID(txCtx, mustBookingID(bookingID))
		if err != nil {
			return err
		}
		saga, err := c.sagaRepo.FindByBookingID(txCtx, bookingID)
		if err != nil {
			return err
		}

		saga.UpdateState(vo.SagaStateFailedTechnical, time.Now())
		if err := c.sagaRepo.Update(txCtx, saga); err != nil {
			return err
		}

		booking.FailTechnical(time.Now())
		return c.bookingRepo.Update(txCtx, booking)
	})
}

// HandleChatRoomCreated is called when Interaction emits ChatRoomCreated.
// Saga: WAITING_FOR_CHAT -> COMPLETED. Booking -> ACCEPTED.
func (c *SagaCoordinator) HandleChatRoomCreated(ctx context.Context, bookingID string, eventID string) error {
	return c.withTx(ctx, func(txCtx context.Context) error {
		alreadyProcessed, err := persistence.CheckAndRecordEvent(txCtx, c.db, eventID, "interaction.chat-room-created.v1")
		if err != nil {
			return err
		}
		if alreadyProcessed {
			return nil
		}

		booking, err := c.bookingRepo.FindByID(txCtx, mustBookingID(bookingID))
		if err != nil {
			return err
		}
		saga, err := c.sagaRepo.FindByBookingID(txCtx, bookingID)
		if err != nil {
			return err
		}

		// Check if companion has any other overlapping booking in ACCEPTED state within the transaction
		companionOverlap, err := c.bookingRepo.HasOverlappingBooking(
			txCtx,
			booking.CompanionID().String(),
			true,
			[]vo.BookingStatus{vo.StatusAccepted},
			booking.TimeRange().StartTime(),
			booking.TimeRange().EndTime(),
		)
		if err != nil {
			return err
		}
		if companionOverlap {
			// Companion accepted another booking in this overlapping period concurrently.
			// Since chat was already created and escrow was successful, we must trigger compensation:
			// 1. Move saga state to REVERTING_ESCROW
			saga.UpdateState(vo.SagaStateRevertingEscrow, time.Now())
			if err := c.sagaRepo.Update(txCtx, saga); err != nil {
				return err
			}

			// 2. Cancel booking aggregate with SYSTEM role to trigger BookingCancelledEarly event
			// which automatically notifies Interaction to lock/close the chat room.
			if err := booking.Cancel(vo.ActorRole("SYSTEM"), time.Now()); err != nil {
				booking.FailTechnical(time.Now())
			}
			if err := c.bookingRepo.Update(txCtx, booking); err != nil {
				return err
			}

			// 3. Publish RefundEscrowCommand to notify Finance to refund the client's money
			return c.outbox.Publish(txCtx, event.RefundEscrowCommand{
				RefundEscrowRequest: &financev1.RefundEscrowRequest{
					BookingId:    bookingID,
					ClientId:     booking.ClientID().String(),
					RefundAmount: booking.Scenario().Price().Amount(),
				},
				Timestamp: time.Now(),
			})
		}

		// [INV-B03] Transition the booking to ACCEPTED
		companionID := booking.CompanionID()
		if err := booking.Accept(companionID, time.Now()); err != nil {
			return err
		}
		if err := c.bookingRepo.Update(txCtx, booking); err != nil {
			return err
		}

		saga.UpdateState(vo.SagaStateCompleted, time.Now())
		return c.sagaRepo.Update(txCtx, saga)
	})
}

// HandleChatRoomFailed is called when Interaction emits ChatRoomCreationFailed.
// Saga: WAITING_FOR_CHAT -> REVERTING_ESCROW. Writes RefundEscrowCommand to outbox.
func (c *SagaCoordinator) HandleChatRoomFailed(ctx context.Context, bookingID string, eventID string) error {
	return c.withTx(ctx, func(txCtx context.Context) error {
		alreadyProcessed, err := persistence.CheckAndRecordEvent(txCtx, c.db, eventID, "interaction.chat-room-creation-failed.v1")
		if err != nil {
			return err
		}
		if alreadyProcessed {
			return nil
		}

		booking, err := c.bookingRepo.FindByID(txCtx, mustBookingID(bookingID))
		if err != nil {
			return err
		}
		saga, err := c.sagaRepo.FindByBookingID(txCtx, bookingID)
		if err != nil {
			return err
		}

		saga.UpdateState(vo.SagaStateRevertingEscrow, time.Now())
		if err := c.sagaRepo.Update(txCtx, saga); err != nil {
			return err
		}

		return c.outbox.Publish(txCtx, event.RefundEscrowCommand{
			RefundEscrowRequest: &financev1.RefundEscrowRequest{
				BookingId:    bookingID,
				ClientId:     booking.ClientID().String(),
				RefundAmount: booking.Scenario().Price().Amount(),
			},
			Timestamp: time.Now(),
		})
	})
}

// HandleRefundSuccess is called when Finance emits RefundSuccess.
// Saga: REVERTING_ESCROW -> FAILED_TECHNICAL. Booking -> CANCELLED.
func (c *SagaCoordinator) HandleRefundSuccess(ctx context.Context, bookingID string, eventID string) error {
	return c.withTx(ctx, func(txCtx context.Context) error {
		alreadyProcessed, err := persistence.CheckAndRecordEvent(txCtx, c.db, eventID, "finance.escrow-refunded.v1")
		if err != nil {
			return err
		}
		if alreadyProcessed {
			return nil
		}

		booking, err := c.bookingRepo.FindByID(txCtx, mustBookingID(bookingID))
		if err != nil {
			return err
		}
		saga, err := c.sagaRepo.FindByBookingID(txCtx, bookingID)
		if err != nil {
			return err
		}

		saga.UpdateState(vo.SagaStateFailedTechnical, time.Now())
		if err := c.sagaRepo.Update(txCtx, saga); err != nil {
			return err
		}

		booking.FailTechnical(time.Now())
		return c.bookingRepo.Update(txCtx, booking)
	})
}

// HandleRefundFailed is called when Finance emits RefundFailed (retry exhausted).
// Saga: REVERTING_ESCROW -> FAILED_REQUIRES_ADMIN. Booking -> CANCELLED.
// Requires manual admin intervention to reconcile the escrow.
func (c *SagaCoordinator) HandleRefundFailed(ctx context.Context, bookingID string, eventID string) error {
	return c.withTx(ctx, func(txCtx context.Context) error {
		alreadyProcessed, err := persistence.CheckAndRecordEvent(txCtx, c.db, eventID, "finance.refund-failed.v1")
		if err != nil {
			return err
		}
		if alreadyProcessed {
			return nil
		}

		booking, err := c.bookingRepo.FindByID(txCtx, mustBookingID(bookingID))
		if err != nil {
			return err
		}
		saga, err := c.sagaRepo.FindByBookingID(txCtx, bookingID)
		if err != nil {
			return err
		}

		// [ALERT] Refund failed after retries — escalate to admin
		log.Printf("[SAGA][ALERT] Refund failed for booking %s. Manual intervention required. SagaID=%s", bookingID, saga.ID)
		saga.UpdateState(vo.SagaStateFailedRequiresAdmin, time.Now())
		if err := c.sagaRepo.Update(txCtx, saga); err != nil {
			return err
		}

		booking.FailTechnical(time.Now())
		return c.bookingRepo.Update(txCtx, booking)
	})
}

// HandleCoinsFrozen is called when Finance emits CoinsFrozen event.
// Transitions the booking status from PENDING_RESERVING to PENDING.
func (c *SagaCoordinator) HandleCoinsFrozen(ctx context.Context, bookingID string, eventID string) error {
	return c.withTx(ctx, func(txCtx context.Context) error {
		alreadyProcessed, err := persistence.CheckAndRecordEvent(txCtx, c.db, eventID, "finance.coins-frozen.v1")
		if err != nil {
			return err
		}
		if alreadyProcessed {
			return nil
		}

		booking, err := c.bookingRepo.FindByID(txCtx, mustBookingID(bookingID))
		if err != nil {
			return err
		}

		if booking.Status() != vo.StatusPendingReserving {
			// If the booking has already been cancelled or rejected, release the frozen coins asynchronously
			if booking.Status() == vo.StatusCancelled || booking.Status() == vo.StatusRejected {
				evt := event.CoinsUnfreezeRequested{
					CoinsUnfreezeRequested: &financev1events.CoinsUnfreezeRequested{
						BookingId:  booking.ID().String(),
						UserId:     booking.ClientID().String(),
						Amount:     booking.Scenario().Price().Amount(),
						OccurredAt: timestamppb.Now(),
					},
					Timestamp: time.Now(),
				}
				if err := c.outbox.Publish(txCtx, evt); err != nil {
					log.Printf("[SAGA] Failed to publish CoinsUnfreezeRequested event for already cancelled/rejected booking %s: %v", booking.ID().String(), err)
				}
			}
			// For all other statuses (PENDING, ACCEPTED, COMPLETED, DISPUTED, RESOLVED), treat as a successful no-op
			return nil
		}

		if err := booking.ConfirmReserved(time.Now()); err != nil {
			return err
		}

		return c.bookingRepo.Update(txCtx, booking)
	})
}

// HandleCoinsFreezeFailed is called when Finance emits CoinsFreezeFailed event.
// Transitions the booking status from PENDING_RESERVING to CANCELLED.
func (c *SagaCoordinator) HandleCoinsFreezeFailed(ctx context.Context, bookingID string, eventID string) error {
	return c.withTx(ctx, func(txCtx context.Context) error {
		alreadyProcessed, err := persistence.CheckAndRecordEvent(txCtx, c.db, eventID, "finance.coins-freeze-failed.v1")
		if err != nil {
			return err
		}
		if alreadyProcessed {
			return nil
		}

		booking, err := c.bookingRepo.FindByID(txCtx, mustBookingID(bookingID))
		if err != nil {
			return err
		}

		if err := booking.CancelReserving("coin freeze failed", time.Now()); err != nil {
			return err
		}

		return c.bookingRepo.Update(txCtx, booking)
	})
}

// withTx wraps fn in a GORM transaction, rolling back on error.
func (c *SagaCoordinator) withTx(ctx context.Context, fn func(txCtx context.Context) error) error {
	tx := c.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}
	txCtx := context.WithValue(ctx, vo.TxKey, tx)
	if err := fn(txCtx); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

func mustBookingID(id string) vo.BookingID {
	bid, err := vo.BookingIDFromString(id)
	if err != nil {
		panic(domainerr.ErrBookingNotFound)
	}
	return bid
}
