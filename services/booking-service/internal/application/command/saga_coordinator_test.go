package command_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/rent-a-girlfriend/booking-service/internal/application/command"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/aggregate"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/vo"
	"github.com/rent-a-girlfriend/booking-service/internal/infrastructure/persistence"
)

func TestSagaCoordinator_HandleEscrowSuccess(t *testing.T) {
	bookingRepo := NewMockBookingRepository()
	sagaRepo := NewMockBookingSagaRepository()
	db := NewMockGormDB()
	outbox := persistence.NewOutboxPublisher(db)

	coordinator := command.NewSagaCoordinator(bookingRepo, sagaRepo, db, outbox)

	clientID, _ := vo.NewClientID("550e8400-e29b-41d4-a716-446655440001")
	companionID, _ := vo.NewCompanionID("550e8400-e29b-41d4-a716-446655440002")
	price := vo.MustMoney(500)
	snap, _ := vo.NewScenarioSnapshot(price, 120)
	now := time.Now()
	tr, _ := vo.NewTimeRange(now.Add(3*time.Hour), now.Add(5*time.Hour))

	bid := vo.NewBookingID()
	b := aggregate.Reconstitute(bid, clientID, companionID, snap, tr, vo.StatusPending, "", false, 1, now, now)
	_ = bookingRepo.Save(context.Background(), b)

	sagaID := uuid.New().String()
	saga := aggregate.NewBookingAcceptSaga(sagaID, bid.String(), now)
	saga.UpdateState(vo.SagaStateWaitingForEscrow, now)
	_ = sagaRepo.Save(context.Background(), saga)

	err := coordinator.HandleEscrowSuccess(context.Background(), bid.String())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify saga state updated to WAITING_FOR_CHAT
	updatedSaga, _ := sagaRepo.FindByBookingID(context.Background(), bid.String())
	if updatedSaga.State != vo.SagaStateWaitingForChat {
		t.Errorf("expected saga state %s, got %s", vo.SagaStateWaitingForChat, updatedSaga.State)
	}
}

func TestSagaCoordinator_HandleEscrowFailed(t *testing.T) {
	bookingRepo := NewMockBookingRepository()
	sagaRepo := NewMockBookingSagaRepository()
	db := NewMockGormDB()
	outbox := persistence.NewOutboxPublisher(db)

	coordinator := command.NewSagaCoordinator(bookingRepo, sagaRepo, db, outbox)

	clientID, _ := vo.NewClientID("550e8400-e29b-41d4-a716-446655440001")
	companionID, _ := vo.NewCompanionID("550e8400-e29b-41d4-a716-446655440002")
	price := vo.MustMoney(500)
	snap, _ := vo.NewScenarioSnapshot(price, 120)
	now := time.Now()
	tr, _ := vo.NewTimeRange(now.Add(3*time.Hour), now.Add(5*time.Hour))

	bid := vo.NewBookingID()
	b := aggregate.Reconstitute(bid, clientID, companionID, snap, tr, vo.StatusPending, "", false, 1, now, now)
	_ = bookingRepo.Save(context.Background(), b)

	sagaID := uuid.New().String()
	saga := aggregate.NewBookingAcceptSaga(sagaID, bid.String(), now)
	saga.UpdateState(vo.SagaStateWaitingForEscrow, now)
	_ = sagaRepo.Save(context.Background(), saga)

	err := coordinator.HandleEscrowFailed(context.Background(), bid.String())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify saga state updated to FAILED_TECHNICAL
	updatedSaga, _ := sagaRepo.FindByBookingID(context.Background(), bid.String())
	if updatedSaga.State != vo.SagaStateFailedTechnical {
		t.Errorf("expected saga state %s, got %s", vo.SagaStateFailedTechnical, updatedSaga.State)
	}

	// Verify booking status updated to CANCELLED/TECHNICAL_FAILED
	updatedBooking, _ := bookingRepo.FindByID(context.Background(), bid)
	if updatedBooking.Status() != vo.StatusCancelled {
		t.Errorf("expected booking status %s, got %s", vo.StatusCancelled, updatedBooking.Status())
	}
}

func TestSagaCoordinator_HandleChatRoomCreated_Success(t *testing.T) {
	bookingRepo := NewMockBookingRepository()
	sagaRepo := NewMockBookingSagaRepository()
	db := NewMockGormDB()
	outbox := persistence.NewOutboxPublisher(db)

	coordinator := command.NewSagaCoordinator(bookingRepo, sagaRepo, db, outbox)

	clientID, _ := vo.NewClientID("550e8400-e29b-41d4-a716-446655440001")
	companionID, _ := vo.NewCompanionID("550e8400-e29b-41d4-a716-446655440002")
	price := vo.MustMoney(500)
	snap, _ := vo.NewScenarioSnapshot(price, 120)
	now := time.Now()
	tr, _ := vo.NewTimeRange(now.Add(3*time.Hour), now.Add(5*time.Hour))

	bid := vo.NewBookingID()
	b := aggregate.Reconstitute(bid, clientID, companionID, snap, tr, vo.StatusPending, "", false, 1, now, now)
	_ = bookingRepo.Save(context.Background(), b)

	sagaID := uuid.New().String()
	saga := aggregate.NewBookingAcceptSaga(sagaID, bid.String(), now)
	saga.UpdateState(vo.SagaStateWaitingForChat, now)
	_ = sagaRepo.Save(context.Background(), saga)

	err := coordinator.HandleChatRoomCreated(context.Background(), bid.String())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify saga state updated to COMPLETED
	updatedSaga, _ := sagaRepo.FindByBookingID(context.Background(), bid.String())
	if updatedSaga.State != vo.SagaStateCompleted {
		t.Errorf("expected saga state %s, got %s", vo.SagaStateCompleted, updatedSaga.State)
	}

	// Verify booking status updated to ACCEPTED
	updatedBooking, _ := bookingRepo.FindByID(context.Background(), bid)
	if updatedBooking.Status() != vo.StatusAccepted {
		t.Errorf("expected booking status %s, got %s", vo.StatusAccepted, updatedBooking.Status())
	}
}

func TestSagaCoordinator_HandleChatRoomCreated_OverlapCompensation(t *testing.T) {
	bookingRepo := NewMockBookingRepository()
	sagaRepo := NewMockBookingSagaRepository()
	db := NewMockGormDB()
	outbox := persistence.NewOutboxPublisher(db)

	coordinator := command.NewSagaCoordinator(bookingRepo, sagaRepo, db, outbox)

	clientID, _ := vo.NewClientID("550e8400-e29b-41d4-a716-446655440001")
	companionID, _ := vo.NewCompanionID("550e8400-e29b-41d4-a716-446655440002")
	price := vo.MustMoney(500)
	snap, _ := vo.NewScenarioSnapshot(price, 120)
	now := time.Now()

	// 1. Existing accepted booking in [3h, 5h]
	tr1, _ := vo.NewTimeRange(now.Add(3*time.Hour), now.Add(5*time.Hour))
	b1 := aggregate.Reconstitute(vo.NewBookingID(), clientID, companionID, snap, tr1, vo.StatusAccepted, "", false, 1, now, now)
	_ = bookingRepo.Save(context.Background(), b1)

	// 2. This saga's booking in [4h, 6h] - overlap!
	tr2, _ := vo.NewTimeRange(now.Add(4*time.Hour), now.Add(6*time.Hour))
	bid2 := vo.NewBookingID()
	b2 := aggregate.Reconstitute(bid2, clientID, companionID, snap, tr2, vo.StatusPending, "", false, 1, now, now)
	_ = bookingRepo.Save(context.Background(), b2)

	sagaID := uuid.New().String()
	saga := aggregate.NewBookingAcceptSaga(sagaID, bid2.String(), now)
	saga.UpdateState(vo.SagaStateWaitingForChat, now)
	_ = sagaRepo.Save(context.Background(), saga)

	// When ChatRoomCreated is handled, it should detect the overlap and trigger compensation!
	err := coordinator.HandleChatRoomCreated(context.Background(), bid2.String())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify saga state updated to REVERTING_ESCROW
	updatedSaga, _ := sagaRepo.FindByBookingID(context.Background(), bid2.String())
	if updatedSaga.State != vo.SagaStateRevertingEscrow {
		t.Errorf("expected saga state %s, got %s", vo.SagaStateRevertingEscrow, updatedSaga.State)
	}

	// Verify booking status updated to CANCELLED
	updatedBooking, _ := bookingRepo.FindByID(context.Background(), bid2)
	if updatedBooking.Status() != vo.StatusCancelled {
		t.Errorf("expected booking status %s, got %s", vo.StatusCancelled, updatedBooking.Status())
	}
}

func TestSagaCoordinator_HandleChatRoomFailed(t *testing.T) {
	bookingRepo := NewMockBookingRepository()
	sagaRepo := NewMockBookingSagaRepository()
	db := NewMockGormDB()
	outbox := persistence.NewOutboxPublisher(db)

	coordinator := command.NewSagaCoordinator(bookingRepo, sagaRepo, db, outbox)

	clientID, _ := vo.NewClientID("550e8400-e29b-41d4-a716-446655440001")
	companionID, _ := vo.NewCompanionID("550e8400-e29b-41d4-a716-446655440002")
	price := vo.MustMoney(500)
	snap, _ := vo.NewScenarioSnapshot(price, 120)
	now := time.Now()
	tr, _ := vo.NewTimeRange(now.Add(3*time.Hour), now.Add(5*time.Hour))

	bid := vo.NewBookingID()
	b := aggregate.Reconstitute(bid, clientID, companionID, snap, tr, vo.StatusPending, "", false, 1, now, now)
	_ = bookingRepo.Save(context.Background(), b)

	sagaID := uuid.New().String()
	saga := aggregate.NewBookingAcceptSaga(sagaID, bid.String(), now)
	saga.UpdateState(vo.SagaStateWaitingForChat, now)
	_ = sagaRepo.Save(context.Background(), saga)

	err := coordinator.HandleChatRoomFailed(context.Background(), bid.String())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify saga state updated to REVERTING_ESCROW
	updatedSaga, _ := sagaRepo.FindByBookingID(context.Background(), bid.String())
	if updatedSaga.State != vo.SagaStateRevertingEscrow {
		t.Errorf("expected saga state %s, got %s", vo.SagaStateRevertingEscrow, updatedSaga.State)
	}
}

func TestSagaCoordinator_HandleRefundSuccess(t *testing.T) {
	bookingRepo := NewMockBookingRepository()
	sagaRepo := NewMockBookingSagaRepository()
	db := NewMockGormDB()
	outbox := persistence.NewOutboxPublisher(db)

	coordinator := command.NewSagaCoordinator(bookingRepo, sagaRepo, db, outbox)

	clientID, _ := vo.NewClientID("550e8400-e29b-41d4-a716-446655440001")
	companionID, _ := vo.NewCompanionID("550e8400-e29b-41d4-a716-446655440002")
	price := vo.MustMoney(500)
	snap, _ := vo.NewScenarioSnapshot(price, 120)
	now := time.Now()
	tr, _ := vo.NewTimeRange(now.Add(3*time.Hour), now.Add(5*time.Hour))

	bid := vo.NewBookingID()
	b := aggregate.Reconstitute(bid, clientID, companionID, snap, tr, vo.StatusPending, "", false, 1, now, now)
	_ = bookingRepo.Save(context.Background(), b)

	sagaID := uuid.New().String()
	saga := aggregate.NewBookingAcceptSaga(sagaID, bid.String(), now)
	saga.UpdateState(vo.SagaStateRevertingEscrow, now)
	_ = sagaRepo.Save(context.Background(), saga)

	err := coordinator.HandleRefundSuccess(context.Background(), bid.String())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify saga state updated to FAILED_TECHNICAL
	updatedSaga, _ := sagaRepo.FindByBookingID(context.Background(), bid.String())
	if updatedSaga.State != vo.SagaStateFailedTechnical {
		t.Errorf("expected saga state %s, got %s", vo.SagaStateFailedTechnical, updatedSaga.State)
	}

	// Verify booking status updated to CANCELLED/TECHNICAL_FAILED
	updatedBooking, _ := bookingRepo.FindByID(context.Background(), bid)
	if updatedBooking.Status() != vo.StatusCancelled {
		t.Errorf("expected booking status %s, got %s", vo.StatusCancelled, updatedBooking.Status())
	}
}

func TestSagaCoordinator_HandleRefundFailed(t *testing.T) {
	bookingRepo := NewMockBookingRepository()
	sagaRepo := NewMockBookingSagaRepository()
	db := NewMockGormDB()
	outbox := persistence.NewOutboxPublisher(db)

	coordinator := command.NewSagaCoordinator(bookingRepo, sagaRepo, db, outbox)

	clientID, _ := vo.NewClientID("550e8400-e29b-41d4-a716-446655440001")
	companionID, _ := vo.NewCompanionID("550e8400-e29b-41d4-a716-446655440002")
	price := vo.MustMoney(500)
	snap, _ := vo.NewScenarioSnapshot(price, 120)
	now := time.Now()
	tr, _ := vo.NewTimeRange(now.Add(3*time.Hour), now.Add(5*time.Hour))

	bid := vo.NewBookingID()
	b := aggregate.Reconstitute(bid, clientID, companionID, snap, tr, vo.StatusPending, "", false, 1, now, now)
	_ = bookingRepo.Save(context.Background(), b)

	sagaID := uuid.New().String()
	saga := aggregate.NewBookingAcceptSaga(sagaID, bid.String(), now)
	saga.UpdateState(vo.SagaStateRevertingEscrow, now)
	_ = sagaRepo.Save(context.Background(), saga)

	err := coordinator.HandleRefundFailed(context.Background(), bid.String())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify saga state updated to FAILED_REQUIRES_ADMIN
	updatedSaga, _ := sagaRepo.FindByBookingID(context.Background(), bid.String())
	if updatedSaga.State != vo.SagaStateFailedRequiresAdmin {
		t.Errorf("expected saga state %s, got %s", vo.SagaStateFailedRequiresAdmin, updatedSaga.State)
	}

	// Verify booking status updated to CANCELLED/TECHNICAL_FAILED
	updatedBooking, _ := bookingRepo.FindByID(context.Background(), bid)
	if updatedBooking.Status() != vo.StatusCancelled {
		t.Errorf("expected booking status %s, got %s", vo.StatusCancelled, updatedBooking.Status())
	}
}
