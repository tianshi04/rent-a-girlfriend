package query_test

import (
	"context"
	"testing"
	"time"

	"github.com/rent-a-girlfriend/booking-service/internal/application/query"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/aggregate"
	"github.com/rent-a-girlfriend/booking-service/internal/domain/vo"
)

func TestListBookings_Success(t *testing.T) {
	repo := NewMockBookingRepository()
	handler := query.NewListBookingsHandler(repo)

	clientID, _ := vo.NewClientID("550e8400-e29b-41d4-a716-446655440001")
	companionID, _ := vo.NewCompanionID("550e8400-e29b-41d4-a716-446655440002")
	price := vo.MustMoney(500)
	snap, _ := vo.NewScenarioSnapshot(price, 120)
	now := time.Now()

	// Seed 3 bookings
	for i := 0; i < 3; i++ {
		bid := vo.NewBookingID()
		tr, _ := vo.NewTimeRange(now.Add(time.Duration(3+i*3)*time.Hour), now.Add(time.Duration(5+i*3)*time.Hour))
		b := aggregate.Reconstitute(bid, clientID, companionID, snap, tr, vo.StatusPending, "", false, 1, now, now)
		_ = repo.Save(context.Background(), b)
	}

	cidStr := clientID.String()
	q := query.ListBookingsQuery{
		ClientID: &cidStr,
		Page:     1,
		PageSize: 10,
	}

	res, err := handler.Handle(context.Background(), q)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if res.Total != 3 {
		t.Errorf("expected 3 total bookings, got %d", res.Total)
	}

	if len(res.Bookings) != 3 {
		t.Errorf("expected 3 returned bookings, got %d", len(res.Bookings))
	}
}

func TestListBookings_Pagination(t *testing.T) {
	repo := NewMockBookingRepository()
	handler := query.NewListBookingsHandler(repo)

	clientID, _ := vo.NewClientID("550e8400-e29b-41d4-a716-446655440001")
	companionID, _ := vo.NewCompanionID("550e8400-e29b-41d4-a716-446655440002")
	price := vo.MustMoney(500)
	snap, _ := vo.NewScenarioSnapshot(price, 120)
	now := time.Now()

	// Seed 5 bookings
	for i := 0; i < 5; i++ {
		bid := vo.NewBookingID()
		tr, _ := vo.NewTimeRange(now.Add(time.Duration(3+i*3)*time.Hour), now.Add(time.Duration(5+i*3)*time.Hour))
		b := aggregate.Reconstitute(bid, clientID, companionID, snap, tr, vo.StatusPending, "", false, 1, now, now)
		_ = repo.Save(context.Background(), b)
	}

	cidStr := clientID.String()
	q := query.ListBookingsQuery{
		ClientID: &cidStr,
		Page:     2,
		PageSize: 2,
	}

	res, err := handler.Handle(context.Background(), q)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if res.Total != 5 {
		t.Errorf("expected 5 total bookings, got %d", res.Total)
	}

	if len(res.Bookings) != 2 {
		t.Errorf("expected 2 paginated bookings on page 2, got %d", len(res.Bookings))
	}
}
