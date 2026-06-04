package query_test

import (
	"context"
	"testing"
	"time"

	"github.com/rent-a-girlfriend/booking-service/internal/application/query"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/aggregate"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/vo"
)

func TestGetBooking_Success(t *testing.T) {
	repo := NewMockBookingRepository()
	handler := query.NewGetBookingHandler(repo)

	clientID, _ := vo.NewClientID("550e8400-e29b-41d4-a716-446655440001")
	companionID, _ := vo.NewCompanionID("550e8400-e29b-41d4-a716-446655440002")
	price := vo.MustMoney(500)
	snap, _ := vo.NewScenarioSnapshot(price, 120)
	now := time.Now()
	tr, _ := vo.NewTimeRange(now.Add(3*time.Hour), now.Add(5*time.Hour))

	bid := vo.NewBookingID()
	b := aggregate.Reconstitute(bid, clientID, companionID, snap, tr, vo.StatusPending, "", false, 1, now, now)
	_ = repo.Save(context.Background(), b)

	q := query.GetBookingQuery{BookingID: bid.String()}
	res, err := handler.Handle(context.Background(), q)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if res.ID() != bid {
		t.Errorf("expected booking ID %s, got %s", bid.String(), res.ID().String())
	}
}

func TestGetBooking_NotFound(t *testing.T) {
	repo := NewMockBookingRepository()
	handler := query.NewGetBookingHandler(repo)

	q := query.GetBookingQuery{BookingID: vo.NewBookingID().String()}
	_, err := handler.Handle(context.Background(), q)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
